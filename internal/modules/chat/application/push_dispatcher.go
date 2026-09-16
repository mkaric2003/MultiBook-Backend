package application

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
	notificationsdomain "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

const (
	maximumPushAttempts       = 8
	stalePushClaimAge         = 5 * time.Minute
	maximumPushBodyRunes      = 120
	maximumStoredFailureRunes = 1000
)

type PushDispatcher struct {
	outbox  PushOutbox
	gateway PushGateway
	now     func() time.Time
}

func NewPushDispatcher(outbox PushOutbox, gateway PushGateway) *PushDispatcher {
	return &PushDispatcher{outbox: outbox, gateway: gateway, now: time.Now}
}

// ProcessNext claims and finalizes at most one job. The boolean is false only
// when no eligible work exists. Failed deliveries are durably scheduled before
// the error is returned to the background runner.
func (d *PushDispatcher) ProcessNext(ctx context.Context) (bool, error) {
	now := d.now().UTC()
	job, err := d.outbox.ClaimNext(ctx, now.Add(-stalePushClaimAge), maximumPushAttempts)
	if err != nil || job == nil {
		return false, err
	}
	event := pushEvent(*job)
	delivered, deliveryErr := d.gateway.DeliverPush(ctx, event)
	if deliveryErr != nil {
		retryAt := now.Add(pushRetryDelay(job.Attempts))
		if markErr := d.outbox.MarkFailed(ctx, job.MessageID, job.Attempts, retryAt, truncateRunes(deliveryErr.Error(), maximumStoredFailureRunes)); markErr != nil {
			return true, fmt.Errorf("mark chat push failed after delivery error %v: %w", deliveryErr, markErr)
		}
		return true, fmt.Errorf("deliver chat push: %w", deliveryErr)
	}
	if !delivered {
		if err := d.outbox.MarkSkipped(ctx, job.MessageID, job.Attempts); err != nil {
			return true, fmt.Errorf("mark chat push skipped: %w", err)
		}
		return true, nil
	}
	if err := d.outbox.MarkSent(ctx, job.MessageID, job.Attempts); err != nil {
		return true, fmt.Errorf("mark chat push sent: %w", err)
	}
	return true, nil
}

func pushEvent(job domain.PushJob) notificationsdomain.Event {
	title := job.BusinessName
	if job.SenderID == job.CustomerID {
		title = job.CustomerName
	}
	businessImage := optionalString(job.BusinessImagePath)
	customerImage := optionalString(job.CustomerImagePath)
	notificationID := "chat_" + job.MessageID.String()
	return notificationsdomain.Event{
		ID: notificationID, RecipientID: job.RecipientID, Kind: "chat_message",
		Title: title, Body: truncateRunes(job.MessageText, maximumPushBodyRunes),
		Data: map[string]string{
			"type": "chat_message", "notificationId": notificationID,
			"conversationId": job.ConversationID.String(), "businessId": job.BusinessID,
			"businessOwnerId": job.BusinessOwnerID, "businessName": job.BusinessName,
			"businessImageUrl": businessImage, "customerId": job.CustomerID,
			"customerName": job.CustomerName, "customerImageUrl": customerImage,
		},
	}
}

func pushRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := 5 * time.Second * time.Duration(1<<min(attempt-1, 8))
	return min(delay, 15*time.Minute)
}

func truncateRunes(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return string(runes[:maximum])
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
