package application

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Queries reads metrics directly from reservation source-of-truth tables.
type Queries interface {
	GetDashboardMetrics(context.Context, string, uuid.UUID, time.Time) (DashboardMetrics, error)
	GetEarningsMetrics(context.Context, string, uuid.UUID, EarningsFilter) (EarningsMetrics, error)
}

// MetricsUpdates publishes committed reservation changes for one business.
// Notifications are invalidation signals; consumers reload the authoritative
// metrics instead of treating an event as the metric value itself.
type MetricsUpdates interface {
	SubscribeChanges(uuid.UUID) (<-chan struct{}, func())
}
