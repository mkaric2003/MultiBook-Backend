package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
	notificationsdomain "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type Cursor struct {
	At time.Time
	ID uuid.UUID
}

type Commands interface {
	GetOrCreate(ctx context.Context, actor Actor, businessID uuid.UUID, customerID string) (domain.Conversation, error)
	SendMessage(ctx context.Context, actorID string, conversationID, messageID uuid.UUID, text string) (domain.Message, error)
	MarkRead(ctx context.Context, actorID string, conversationID uuid.UUID) error
	SetTyping(ctx context.Context, actorID string, conversationID uuid.UUID, until *time.Time) error
	SetActive(ctx context.Context, actorID string, conversationID uuid.UUID, until *time.Time) error
}

type Queries interface {
	Get(ctx context.Context, actorID string, conversationID uuid.UUID) (domain.Conversation, error)
	List(ctx context.Context, actorID string, cursor *Cursor, limit int) ([]domain.Conversation, error)
	ListMessages(ctx context.Context, actorID string, conversationID uuid.UUID, cursor *Cursor, limit int) ([]domain.Message, error)
	UnreadCount(ctx context.Context, actorID string) (int, error)
}

// ChatUpdates publishes committed conversation invalidations. Events carry no
// message content; consumers reload authorized state through Queries.
type ChatUpdates interface {
	SubscribeChanges(actorID string) (<-chan Change, func())
}

type Change struct {
	ConversationID *uuid.UUID
}

type PushOutbox interface {
	ClaimNext(ctx context.Context, staleBefore time.Time, maximumAttempts int) (*domain.PushJob, error)
	MarkSent(ctx context.Context, messageID uuid.UUID, attempt int) error
	MarkSkipped(ctx context.Context, messageID uuid.UUID, attempt int) error
	MarkFailed(ctx context.Context, messageID uuid.UUID, attempt int, retryAt time.Time, failure string) error
}

type PushGateway interface {
	DeliverPush(ctx context.Context, event notificationsdomain.Event) (bool, error)
}
