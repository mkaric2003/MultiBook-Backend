package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
)

const chatChangesChannel = "chat_changed"

// ChatUpdateListener uses one dedicated PostgreSQL connection to serve every
// local SSE subscriber. Subscribers are indexed by authenticated participant ID, so
// unrelated chat activity is discarded before reaching HTTP handlers.
type ChatUpdateListener struct {
	databaseURL string
	logger      *slog.Logger

	mu             sync.RWMutex
	nextSubscriber uint64
	subscribers    map[string]map[uint64]chan application.Change
}

func NewChatUpdateListener(databaseURL string, logger *slog.Logger) *ChatUpdateListener {
	return &ChatUpdateListener{
		databaseURL: databaseURL,
		logger:      logger,
		subscribers: make(map[string]map[uint64]chan application.Change),
	}
}

var _ application.ChatUpdates = (*ChatUpdateListener)(nil)

func (l *ChatUpdateListener) Run(ctx context.Context) {
	if l.databaseURL == "" {
		return
	}
	go l.listen(ctx)
}

func (l *ChatUpdateListener) SubscribeChanges(actorID string) (<-chan application.Change, func()) {
	updates := make(chan application.Change, 1)
	l.mu.Lock()
	l.nextSubscriber++
	id := l.nextSubscriber
	if l.subscribers[actorID] == nil {
		l.subscribers[actorID] = make(map[uint64]chan application.Change)
	}
	l.subscribers[actorID][id] = updates
	l.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			l.mu.Lock()
			delete(l.subscribers[actorID], id)
			if len(l.subscribers[actorID]) == 0 {
				delete(l.subscribers, actorID)
			}
			l.mu.Unlock()
		})
	}
	return updates, unsubscribe
}

type chatChangePayload struct {
	ConversationID uuid.UUID `json:"conversationId"`
	ParticipantIDs []string  `json:"participantIds"`
}

func (l *ChatUpdateListener) listen(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		connection, err := pgx.Connect(ctx, l.databaseURL)
		if err == nil {
			_, err = connection.Exec(ctx, "LISTEN "+chatChangesChannel)
		}
		if err != nil {
			if connection != nil {
				_ = connection.Close(ctx)
			}
			l.logger.Warn("chat listener connection failed", "error", err)
			if !waitForChatRetry(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		backoff = time.Second
		l.publishSync()
		for ctx.Err() == nil {
			notification, waitErr := connection.WaitForNotification(ctx)
			if waitErr != nil {
				err = waitErr
				break
			}
			var payload chatChangePayload
			if decodeErr := json.Unmarshal([]byte(notification.Payload), &payload); decodeErr != nil || payload.ConversationID == uuid.Nil || len(payload.ParticipantIDs) != 2 {
				l.logger.Warn("chat listener ignored invalid notification payload")
				continue
			}
			l.publish(payload)
		}
		_ = connection.Close(context.Background())
		if ctx.Err() == nil {
			l.logger.Warn("chat listener disconnected", "error", err)
		}
	}
}

func (l *ChatUpdateListener) publish(payload chatChangePayload) {
	conversationID := payload.ConversationID
	change := application.Change{ConversationID: &conversationID}
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, actorID := range payload.ParticipantIDs {
		for _, subscriber := range l.subscribers[actorID] {
			offerChatChange(subscriber, change)
		}
	}
}

func (l *ChatUpdateListener) publishSync() {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, actorSubscribers := range l.subscribers {
		for _, subscriber := range actorSubscribers {
			offerChatChange(subscriber, application.Change{})
		}
	}
}

// A full channel must not silently lose a different conversation ID. Replace
// its buffered item with a general sync event, which tells the client to
// reload every currently relevant chat snapshot.
func offerChatChange(subscriber chan application.Change, change application.Change) {
	select {
	case subscriber <- change:
		return
	default:
	}
	select {
	case <-subscriber:
	default:
	}
	select {
	case subscriber <- application.Change{}:
	default:
	}
}

func waitForChatRetry(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
