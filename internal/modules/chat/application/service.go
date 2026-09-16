// Package application contains chat use cases and ports.
package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
)

var (
	ErrNotFound   = errors.New("conversation not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
	ErrConflict   = errors.New("message id conflict")
)

const (
	defaultPageLimit = 30
	maximumPageLimit = 60
	maximumTextRunes = 4000
)

type Actor struct {
	ID   string
	Role string
}

type ConversationPage struct {
	Items      []domain.Conversation
	NextCursor *Cursor
}

type MessagePage struct {
	Items      []domain.Message
	NextCursor *Cursor
}

type Service struct {
	commands Commands
	queries  Queries
	updates  ChatUpdates
}

func NewService(commands Commands, queries Queries, updates ChatUpdates) *Service {
	return &Service{commands: commands, queries: queries, updates: updates}
}

func (s *Service) OpenUpdates(actor Actor) (<-chan Change, func(), error) {
	if actor.ID == "" {
		return nil, nil, ErrForbidden
	}
	if s.updates == nil {
		return nil, nil, errors.New("chat updates are unavailable")
	}
	updates, unsubscribe := s.updates.SubscribeChanges(actor.ID)
	return updates, unsubscribe, nil
}

// CanViewChange prevents a process-wide PostgreSQL invalidation from crossing
// participant boundaries before it is emitted to an SSE client.
func (s *Service) CanViewChange(ctx context.Context, actor Actor, conversationID uuid.UUID) (bool, error) {
	if actor.ID == "" || conversationID == uuid.Nil {
		return false, ErrForbidden
	}
	_, err := s.queries.Get(ctx, actor.ID, conversationID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) GetOrCreate(ctx context.Context, actor Actor, businessID uuid.UUID, requestedCustomerID string) (domain.Conversation, error) {
	if actor.ID == "" || businessID == uuid.Nil {
		return domain.Conversation{}, ErrValidation
	}
	customerID := strings.TrimSpace(requestedCustomerID)
	switch actor.Role {
	case "customer":
		if customerID != "" && customerID != actor.ID {
			return domain.Conversation{}, ErrForbidden
		}
		customerID = actor.ID
	case "provider":
		if customerID == "" || customerID == actor.ID {
			return domain.Conversation{}, ErrValidation
		}
	default:
		return domain.Conversation{}, ErrForbidden
	}
	return s.commands.GetOrCreate(ctx, actor, businessID, customerID)
}

func (s *Service) Get(ctx context.Context, actor Actor, conversationID uuid.UUID) (domain.Conversation, error) {
	if err := validConversationRequest(actor, conversationID); err != nil {
		return domain.Conversation{}, err
	}
	return s.queries.Get(ctx, actor.ID, conversationID)
}

func (s *Service) List(ctx context.Context, actor Actor, cursor *Cursor, limit int) (ConversationPage, error) {
	if actor.ID == "" {
		return ConversationPage{}, ErrForbidden
	}
	limit = pageLimit(limit)
	items, err := s.queries.List(ctx, actor.ID, cursor, limit+1)
	if err != nil {
		return ConversationPage{}, err
	}
	page := ConversationPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		at := last.CreatedAt
		if last.LastMessageAt != nil {
			at = *last.LastMessageAt
		}
		page.NextCursor = &Cursor{At: at, ID: last.ID}
	}
	return page, nil
}

func (s *Service) ListMessages(ctx context.Context, actor Actor, conversationID uuid.UUID, cursor *Cursor, limit int) (MessagePage, error) {
	if err := validConversationRequest(actor, conversationID); err != nil {
		return MessagePage{}, err
	}
	limit = pageLimit(limit)
	items, err := s.queries.ListMessages(ctx, actor.ID, conversationID, cursor, limit+1)
	if err != nil {
		return MessagePage{}, err
	}
	page := MessagePage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{At: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

func (s *Service) SendMessage(ctx context.Context, actor Actor, conversationID, messageID uuid.UUID, text string) (domain.Message, error) {
	if err := validConversationRequest(actor, conversationID); err != nil {
		return domain.Message{}, err
	}
	if messageID == uuid.Nil {
		return domain.Message{}, ErrValidation
	}
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > maximumTextRunes || containsForbiddenControl(text) {
		return domain.Message{}, ErrValidation
	}
	return s.commands.SendMessage(ctx, actor.ID, conversationID, messageID, text)
}

func (s *Service) MarkRead(ctx context.Context, actor Actor, conversationID uuid.UUID) error {
	if err := validConversationRequest(actor, conversationID); err != nil {
		return err
	}
	return s.commands.MarkRead(ctx, actor.ID, conversationID)
}

func (s *Service) SetTyping(ctx context.Context, actor Actor, conversationID uuid.UUID, active bool) error {
	if err := validConversationRequest(actor, conversationID); err != nil {
		return err
	}
	var until *time.Time
	if active {
		value := time.Now().UTC().Add(4 * time.Second)
		until = &value
	}
	return s.commands.SetTyping(ctx, actor.ID, conversationID, until)
}

func (s *Service) SetActive(ctx context.Context, actor Actor, conversationID uuid.UUID, active bool) error {
	if err := validConversationRequest(actor, conversationID); err != nil {
		return err
	}
	var until *time.Time
	if active {
		value := time.Now().UTC().Add(45 * time.Second)
		until = &value
	}
	return s.commands.SetActive(ctx, actor.ID, conversationID, until)
}

func (s *Service) UnreadCount(ctx context.Context, actor Actor) (int, error) {
	if actor.ID == "" {
		return 0, ErrForbidden
	}
	return s.queries.UnreadCount(ctx, actor.ID)
}

func validConversationRequest(actor Actor, conversationID uuid.UUID) error {
	if actor.ID == "" {
		return ErrForbidden
	}
	if conversationID == uuid.Nil {
		return ErrValidation
	}
	return nil
}

func pageLimit(limit int) int {
	if limit < 1 {
		return defaultPageLimit
	}
	if limit > maximumPageLimit {
		return maximumPageLimit
	}
	return limit
}

func containsForbiddenControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) && character != '\n' && character != '\t' {
			return true
		}
	}
	return false
}
