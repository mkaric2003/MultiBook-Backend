package postgres

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
)

const metricsChangesChannel = "business_metrics_changed"

// MetricsUpdateListener turns PostgreSQL commit notifications into local
// invalidation signals. One dedicated connection serves every SSE client, so
// live dashboards do not consume one database connection each.
type MetricsUpdateListener struct {
	databaseURL string
	logger      *slog.Logger

	mu             sync.RWMutex
	nextSubscriber uint64
	subscribers    map[uuid.UUID]map[uint64]chan struct{}
}

func NewMetricsUpdateListener(databaseURL string, logger *slog.Logger) *MetricsUpdateListener {
	return &MetricsUpdateListener{
		databaseURL: databaseURL,
		logger:      logger,
		subscribers: make(map[uuid.UUID]map[uint64]chan struct{}),
	}
}

var _ application.MetricsUpdates = (*MetricsUpdateListener)(nil)

func (l *MetricsUpdateListener) Run(ctx context.Context) {
	if l.databaseURL == "" {
		return
	}
	go l.listen(ctx)
}

func (l *MetricsUpdateListener) SubscribeChanges(businessID uuid.UUID) (<-chan struct{}, func()) {
	updates := make(chan struct{}, 1)
	l.mu.Lock()
	l.nextSubscriber++
	id := l.nextSubscriber
	if l.subscribers[businessID] == nil {
		l.subscribers[businessID] = make(map[uint64]chan struct{})
	}
	l.subscribers[businessID][id] = updates
	l.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			l.mu.Lock()
			delete(l.subscribers[businessID], id)
			if len(l.subscribers[businessID]) == 0 {
				delete(l.subscribers, businessID)
			}
			l.mu.Unlock()
		})
	}
	return updates, cancel
}

func (l *MetricsUpdateListener) listen(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		connection, err := pgx.Connect(ctx, l.databaseURL)
		if err == nil {
			_, err = connection.Exec(ctx, "LISTEN "+metricsChangesChannel)
		}
		if err != nil {
			if connection != nil {
				_ = connection.Close(ctx)
			}
			l.logger.Warn("metrics listener connection failed", "error", err)
			if !waitForRetry(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		backoff = time.Second
		l.publishAll()
		for ctx.Err() == nil {
			notification, waitErr := connection.WaitForNotification(ctx)
			if waitErr != nil {
				err = waitErr
				break
			}
			businessID, parseErr := uuid.Parse(notification.Payload)
			if parseErr != nil {
				l.logger.Warn("metrics listener ignored invalid business id", "payload", notification.Payload)
				continue
			}
			l.publish(businessID)
		}
		_ = connection.Close(context.Background())
		if ctx.Err() == nil {
			l.logger.Warn("metrics listener disconnected", "error", err)
		}
	}
}

func (l *MetricsUpdateListener) publish(businessID uuid.UUID) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, subscriber := range l.subscribers[businessID] {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}

func (l *MetricsUpdateListener) publishAll() {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, businessSubscribers := range l.subscribers {
		for _, subscriber := range businessSubscribers {
			select {
			case subscriber <- struct{}{}:
			default:
			}
		}
	}
}

func waitForRetry(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
