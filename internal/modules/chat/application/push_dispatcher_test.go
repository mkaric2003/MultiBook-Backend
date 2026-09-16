package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
	notificationsdomain "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type pushOutboxStub struct {
	job              *domain.PushJob
	claimStaleBefore time.Time
	claimMaximum     int
	marked           string
	retryAt          time.Time
	failure          string
	err              error
}

func (s *pushOutboxStub) ClaimNext(_ context.Context, staleBefore time.Time, maximum int) (*domain.PushJob, error) {
	s.claimStaleBefore, s.claimMaximum = staleBefore, maximum
	return s.job, s.err
}
func (s *pushOutboxStub) MarkSent(context.Context, uuid.UUID, int) error {
	s.marked = "sent"
	return s.err
}
func (s *pushOutboxStub) MarkSkipped(context.Context, uuid.UUID, int) error {
	s.marked = "skipped"
	return s.err
}
func (s *pushOutboxStub) MarkFailed(_ context.Context, _ uuid.UUID, _ int, retryAt time.Time, failure string) error {
	s.marked, s.retryAt, s.failure = "failed", retryAt, failure
	return s.err
}

type pushGatewayStub struct {
	delivered bool
	event     notificationsdomain.Event
	err       error
}

func (s *pushGatewayStub) DeliverPush(_ context.Context, event notificationsdomain.Event) (bool, error) {
	s.event = event
	return s.delivered, s.err
}

func TestPushDispatcherMarksDeliveredChatPushSent(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	job := pushJobFixture()
	outbox := &pushOutboxStub{job: &job}
	gateway := &pushGatewayStub{delivered: true}
	dispatcher := NewPushDispatcher(outbox, gateway)
	dispatcher.now = func() time.Time { return now }

	processed, err := dispatcher.ProcessNext(context.Background())
	if err != nil || !processed || outbox.marked != "sent" {
		t.Fatalf("processed = %v, marked = %q, error = %v", processed, outbox.marked, err)
	}
	if outbox.claimMaximum != maximumPushAttempts || !outbox.claimStaleBefore.Equal(now.Add(-stalePushClaimAge)) {
		t.Fatalf("claim = %d/%v", outbox.claimMaximum, outbox.claimStaleBefore)
	}
	if gateway.event.Title != job.CustomerName || gateway.event.RecipientID != job.RecipientID || gateway.event.Data["conversationId"] != job.ConversationID.String() {
		t.Fatalf("event = %#v", gateway.event)
	}
}

func TestPushDispatcherMarksMissingDevicesSkipped(t *testing.T) {
	job := pushJobFixture()
	outbox := &pushOutboxStub{job: &job}
	dispatcher := NewPushDispatcher(outbox, &pushGatewayStub{})
	processed, err := dispatcher.ProcessNext(context.Background())
	if err != nil || !processed || outbox.marked != "skipped" {
		t.Fatalf("processed = %v, marked = %q, error = %v", processed, outbox.marked, err)
	}
}

func TestPushDispatcherSchedulesFailedDelivery(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	job := pushJobFixture()
	job.Attempts = 3
	outbox := &pushOutboxStub{job: &job}
	gateway := &pushGatewayStub{err: errors.New(strings.Repeat("failure", 300))}
	dispatcher := NewPushDispatcher(outbox, gateway)
	dispatcher.now = func() time.Time { return now }

	processed, err := dispatcher.ProcessNext(context.Background())
	if !processed || err == nil || outbox.marked != "failed" {
		t.Fatalf("processed = %v, marked = %q, error = %v", processed, outbox.marked, err)
	}
	if !outbox.retryAt.Equal(now.Add(20*time.Second)) || len([]rune(outbox.failure)) != maximumStoredFailureRunes {
		t.Fatalf("retry = %v, failure runes = %d", outbox.retryAt, len([]rune(outbox.failure)))
	}
}

func TestPushDispatcherReturnsIdleWithoutCallingGateway(t *testing.T) {
	processed, err := NewPushDispatcher(&pushOutboxStub{}, &pushGatewayStub{}).ProcessNext(context.Background())
	if err != nil || processed {
		t.Fatalf("processed = %v, error = %v", processed, err)
	}
}

func pushJobFixture() domain.PushJob {
	return domain.PushJob{
		MessageID: uuid.New(), RecipientID: "provider-1", Attempts: 1,
		ConversationID: uuid.New(), BusinessID: uuid.NewString(), BusinessOwnerID: "provider-1",
		BusinessName: "Business", CustomerID: "customer-1", CustomerName: "Customer",
		SenderID: "customer-1", MessageText: "Hello",
	}
}
