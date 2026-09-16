package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
)

type commandStub struct {
	conversation domain.Conversation
	message      domain.Message
	err          error
	customerID   string
	text         string
	typingUntil  *time.Time
	activeUntil  *time.Time
}

func (s *commandStub) GetOrCreate(_ context.Context, _ Actor, _ uuid.UUID, customerID string) (domain.Conversation, error) {
	s.customerID = customerID
	return s.conversation, s.err
}
func (s *commandStub) SendMessage(_ context.Context, _ string, _, _ uuid.UUID, text string) (domain.Message, error) {
	s.text = text
	return s.message, s.err
}
func (s *commandStub) MarkRead(context.Context, string, uuid.UUID) error { return s.err }
func (s *commandStub) SetTyping(_ context.Context, _ string, _ uuid.UUID, until *time.Time) error {
	s.typingUntil = until
	return s.err
}
func (s *commandStub) SetActive(_ context.Context, _ string, _ uuid.UUID, until *time.Time) error {
	s.activeUntil = until
	return s.err
}

type queryStub struct {
	conversations []domain.Conversation
	messages      []domain.Message
	err           error
}

func (s *queryStub) Get(context.Context, string, uuid.UUID) (domain.Conversation, error) {
	return domain.Conversation{}, s.err
}
func (s *queryStub) List(context.Context, string, *Cursor, int) ([]domain.Conversation, error) {
	return s.conversations, s.err
}
func (s *queryStub) ListMessages(context.Context, string, uuid.UUID, *Cursor, int) ([]domain.Message, error) {
	return s.messages, s.err
}
func (s *queryStub) UnreadCount(context.Context, string) (int, error) { return 3, s.err }

type updatesStub struct {
	actorID string
	updates chan Change
}

func (s *updatesStub) SubscribeChanges(actorID string) (<-chan Change, func()) {
	s.actorID = actorID
	return s.updates, func() {}
}

func TestGetOrCreateDerivesCustomerFromAuthenticatedActor(t *testing.T) {
	commands := &commandStub{}
	service := NewService(commands, &queryStub{}, nil)
	_, err := service.GetOrCreate(context.Background(), Actor{ID: "customer-1", Role: "customer"}, uuid.New(), "")
	if err != nil || commands.customerID != "customer-1" {
		t.Fatalf("customer id = %q, error = %v", commands.customerID, err)
	}
	_, err = service.GetOrCreate(context.Background(), Actor{ID: "customer-1", Role: "customer"}, uuid.New(), "another-customer")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want forbidden", err)
	}
}

func TestGetOrCreateRequiresProviderCustomer(t *testing.T) {
	service := NewService(&commandStub{}, &queryStub{}, nil)
	_, err := service.GetOrCreate(context.Background(), Actor{ID: "provider-1", Role: "provider"}, uuid.New(), "")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
}

func TestSendMessageNormalizesAndValidatesText(t *testing.T) {
	commands := &commandStub{}
	service := NewService(commands, &queryStub{}, nil)
	actor := Actor{ID: "customer-1", Role: "customer"}
	_, err := service.SendMessage(context.Background(), actor, uuid.New(), uuid.New(), "  hello\nworld  ")
	if err != nil || commands.text != "hello\nworld" {
		t.Fatalf("text = %q, error = %v", commands.text, err)
	}
	for _, invalid := range []string{"   ", "hello\u0000", strings.Repeat("a", maximumTextRunes+1)} {
		if _, err := service.SendMessage(context.Background(), actor, uuid.New(), uuid.New(), invalid); !errors.Is(err, ErrValidation) {
			t.Fatalf("text %q: error = %v, want validation", invalid[:min(len(invalid), 10)], err)
		}
	}
}

func TestConversationPaginationUsesLastReturnedItem(t *testing.T) {
	now := time.Now().UTC()
	queries := &queryStub{conversations: []domain.Conversation{
		{ID: uuid.New(), CreatedAt: now},
		{ID: uuid.New(), CreatedAt: now.Add(-time.Minute)},
		{ID: uuid.New(), CreatedAt: now.Add(-2 * time.Minute)},
	}}
	service := NewService(&commandStub{}, queries, nil)
	page, err := service.List(context.Background(), Actor{ID: "customer-1"}, nil, 2)
	if err != nil || len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("page = %#v, error = %v", page, err)
	}
	if page.NextCursor.ID != page.Items[1].ID || !page.NextCursor.At.Equal(page.Items[1].CreatedAt) {
		t.Fatalf("cursor = %#v, last = %#v", page.NextCursor, page.Items[1])
	}
}

func TestParticipantExpiryUsesServerTime(t *testing.T) {
	commands := &commandStub{}
	service := NewService(commands, &queryStub{}, nil)
	before := time.Now().UTC()
	if err := service.SetTyping(context.Background(), Actor{ID: "customer-1"}, uuid.New(), true); err != nil {
		t.Fatal(err)
	}
	if commands.typingUntil == nil || commands.typingUntil.Before(before.Add(3*time.Second)) {
		t.Fatalf("typing expiry = %v", commands.typingUntil)
	}
	if err := service.SetActive(context.Background(), Actor{ID: "customer-1"}, uuid.New(), false); err != nil {
		t.Fatal(err)
	}
	if commands.activeUntil != nil {
		t.Fatalf("inactive expiry = %v, want nil", commands.activeUntil)
	}
}

func TestOpenUpdatesScopesSubscriptionToAuthenticatedActor(t *testing.T) {
	updates := &updatesStub{updates: make(chan Change, 1)}
	service := NewService(&commandStub{}, &queryStub{}, updates)
	stream, unsubscribe, err := service.OpenUpdates(Actor{ID: "customer-1"})
	if err != nil || stream == nil || unsubscribe == nil || updates.actorID != "customer-1" {
		t.Fatalf("stream = %v, actor = %q, error = %v", stream, updates.actorID, err)
	}
}
