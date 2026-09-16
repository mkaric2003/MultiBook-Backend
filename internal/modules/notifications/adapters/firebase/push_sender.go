package firebase

import (
	"context"

	"firebase.google.com/go/v4/messaging"
	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type PushSender struct{ client *messaging.Client }

func NewPushSender(client *messaging.Client) *PushSender { return &PushSender{client: client} }
func (s *PushSender) Send(ctx context.Context, tokens []string, event domain.Event) error {
	message := &messaging.MulticastMessage{
		Tokens: tokens, Notification: &messaging.Notification{Title: event.Title, Body: event.Body}, Data: event.Data,
	}
	if event.Kind == "chat_message" {
		message.Android = &messaging.AndroidConfig{Priority: "high", CollapseKey: event.ID}
		message.APNS = &messaging.APNSConfig{
			Headers: map[string]string{"apns-collapse-id": event.ID},
			Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Sound: "default", ContentAvailable: true}},
		}
	}
	_, err := s.client.SendEachForMulticast(ctx, message)
	return err
}
