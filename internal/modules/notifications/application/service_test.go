package application

import (
	"context"
	"testing"

	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type notificationCommandsStub struct{ created bool }

func (s *notificationCommandsStub) Create(context.Context, domain.Event) error {
	s.created = true
	return nil
}
func (s *notificationCommandsStub) MarkRead(context.Context, string, string) error { return nil }
func (s *notificationCommandsStub) UpsertDevice(context.Context, string, string, string) error {
	return nil
}
func (s *notificationCommandsStub) DeleteDevice(context.Context, string, string) error { return nil }

type notificationQueriesStub struct{ tokens []string }

func (s *notificationQueriesStub) List(context.Context, string, ListInput) (Page, error) {
	return Page{}, nil
}
func (s *notificationQueriesStub) UnreadCount(context.Context, string) (int, error) { return 0, nil }
func (s *notificationQueriesStub) DeviceTokens(context.Context, string) ([]string, error) {
	return s.tokens, nil
}

type pushSenderStub struct{ calls int }

func (s *pushSenderStub) Send(context.Context, []string, domain.Event) error {
	s.calls++
	return nil
}

func TestDeliverPushDoesNotCreateInAppNotification(t *testing.T) {
	commands := &notificationCommandsStub{}
	push := &pushSenderStub{}
	service := NewService(commands, &notificationQueriesStub{tokens: []string{"token"}}, push)
	delivered, err := service.DeliverPush(context.Background(), domain.Event{
		ID: "chat-message", RecipientID: "recipient", Kind: "chat_message",
	})
	if err != nil || !delivered || push.calls != 1 || commands.created {
		t.Fatalf("delivered = %v, push calls = %d, created = %v, error = %v", delivered, push.calls, commands.created, err)
	}
}

func TestDeliverPushSkipsRecipientWithoutDevices(t *testing.T) {
	push := &pushSenderStub{}
	service := NewService(&notificationCommandsStub{}, &notificationQueriesStub{}, push)
	delivered, err := service.DeliverPush(context.Background(), domain.Event{
		ID: "chat-message", RecipientID: "recipient", Kind: "chat_message",
	})
	if err != nil || delivered || push.calls != 0 {
		t.Fatalf("delivered = %v, push calls = %d, error = %v", delivered, push.calls, err)
	}
}
