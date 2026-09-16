package postgres

import (
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
)

func TestChatUpdateListenerPublishesOnlyToParticipants(t *testing.T) {
	listener := NewChatUpdateListener("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	first, unsubscribeFirst := listener.SubscribeChanges("first")
	defer unsubscribeFirst()
	second, unsubscribeSecond := listener.SubscribeChanges("second")
	defer unsubscribeSecond()
	other, unsubscribeOther := listener.SubscribeChanges("other")
	defer unsubscribeOther()

	conversationID := uuid.New()
	listener.publish(chatChangePayload{ConversationID: conversationID, ParticipantIDs: []string{"first", "second"}})
	for name, updates := range map[string]<-chan application.Change{"first": first, "second": second} {
		select {
		case change := <-updates:
			if change.ConversationID == nil || *change.ConversationID != conversationID {
				t.Fatalf("%s change = %#v", name, change)
			}
		default:
			t.Fatalf("%s did not receive change", name)
		}
	}
	select {
	case change := <-other:
		t.Fatalf("unrelated subscriber received %#v", change)
	default:
	}
}

func TestChatUpdateListenerPublishesSyncToAllSubscribers(t *testing.T) {
	listener := NewChatUpdateListener("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	updates, unsubscribe := listener.SubscribeChanges("first")
	defer unsubscribe()
	listener.publishSync()
	select {
	case change := <-updates:
		if change.ConversationID != nil {
			t.Fatalf("sync change = %#v", change)
		}
	default:
		t.Fatal("subscriber did not receive sync")
	}
}

func TestChatUpdateListenerCoalescesOverflowIntoFullSync(t *testing.T) {
	listener := NewChatUpdateListener("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	updates, unsubscribe := listener.SubscribeChanges("first")
	defer unsubscribe()
	listener.publish(chatChangePayload{ConversationID: uuid.New(), ParticipantIDs: []string{"first", "second"}})
	listener.publish(chatChangePayload{ConversationID: uuid.New(), ParticipantIDs: []string{"first", "second"}})
	select {
	case change := <-updates:
		if change.ConversationID != nil {
			t.Fatalf("overflow change = %#v, want full sync", change)
		}
	default:
		t.Fatal("subscriber did not receive overflow sync")
	}
}
