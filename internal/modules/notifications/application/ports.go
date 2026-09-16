package application

import (
	"context"

	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type NotificationCommands interface {
	Create(ctx context.Context, event domain.Event) error
	MarkRead(ctx context.Context, recipientID, notificationID string) error
	UpsertDevice(ctx context.Context, recipientID, deviceID, token string) error
	DeleteDevice(ctx context.Context, recipientID, deviceID string) error
}

type NotificationQueries interface {
	List(ctx context.Context, recipientID string, input ListInput) (Page, error)
	UnreadCount(ctx context.Context, recipientID string) (int, error)
	DeviceTokens(ctx context.Context, recipientID string) ([]string, error)
}

type PushSender interface {
	Send(ctx context.Context, tokens []string, event domain.Event) error
}

// Publisher is consumed by booking and appointment use cases after successful
// domain transitions. Notification failures never undo those transitions.
type Publisher interface {
	Publish(ctx context.Context, event domain.Event)
}

// PushPublisher sends a transport-only push without creating an in-app
// notification. The boolean is false when the recipient has no active device.
type PushPublisher interface {
	DeliverPush(ctx context.Context, event domain.Event) (bool, error)
}
