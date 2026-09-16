//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	chatpostgres "github.com/mkaric2003/multibook-backend/internal/modules/chat/adapters/postgres"
	chatapp "github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
)

func TestChatPersistenceOwnershipUnreadAndIdempotency(t *testing.T) {
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	owner := "chat-owner-" + uuid.NewString()
	otherOwner := "chat-other-owner-" + uuid.NewString()
	customer := "chat-customer-" + uuid.NewString()
	otherCustomer := "chat-other-customer-" + uuid.NewString()
	businessID := uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM chat_conversations WHERE business_id=$1`, businessID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM businesses WHERE id=$1`, businessID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=ANY($1)`, []string{owner, otherOwner, customer, otherCustomer})
	}()
	exec(`INSERT INTO users(id,email,role,display_name,business_currency) VALUES
		($1,$1||'@example.test','provider','Owner','BAM'),
		($2,$2||'@example.test','provider','Other owner','BAM'),
		($3,$3||'@example.test','customer','Customer','BAM'),
		($4,$4||'@example.test','customer','Other customer','BAM')`, owner, otherOwner, customer, otherCustomer)
	exec(`INSERT INTO businesses(id,owner_id,type,status,name,name_normalized,category_id,currency)
		VALUES($1,$2,'stay','active','Chat stay','chat stay','hotel','BAM')`, businessID, owner)

	commands := chatpostgres.NewCommandRepository(pool)
	queries := chatpostgres.NewQueries(pool)
	service := chatapp.NewService(commands, queries, nil)
	customerActor := chatapp.Actor{ID: customer, Role: "customer"}
	providerActor := chatapp.Actor{ID: owner, Role: "provider"}

	conversation, err := service.GetOrCreate(ctx, customerActor, businessID, "")
	if err != nil {
		t.Fatal(err)
	}
	same, err := service.GetOrCreate(ctx, providerActor, businessID, customer)
	if err != nil || same.ID != conversation.ID {
		t.Fatalf("idempotent conversation = %#v, %v", same, err)
	}
	if _, err := service.GetOrCreate(ctx, chatapp.Actor{ID: otherOwner, Role: "provider"}, businessID, customer); !errors.Is(err, chatapp.ErrForbidden) {
		t.Fatalf("foreign provider error = %v", err)
	}
	listener, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close(context.Background())
	if _, err := listener.Exec(ctx, "LISTEN chat_changed"); err != nil {
		t.Fatal(err)
	}

	messageID := uuid.New()
	message, err := service.SendMessage(ctx, customerActor, conversation.ID, messageID, "  Hello  ")
	if err != nil || message.Text != "Hello" {
		t.Fatalf("message = %#v, %v", message, err)
	}
	notification, err := listener.WaitForNotification(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var notificationPayload struct {
		ConversationID uuid.UUID `json:"conversationId"`
		ParticipantIDs []string  `json:"participantIds"`
	}
	if err := json.Unmarshal([]byte(notification.Payload), &notificationPayload); err != nil {
		t.Fatal(err)
	}
	if notificationPayload.ConversationID != conversation.ID || len(notificationPayload.ParticipantIDs) != 2 {
		t.Fatalf("notification payload = %#v", notificationPayload)
	}
	replayed, err := service.SendMessage(ctx, customerActor, conversation.ID, messageID, "Hello")
	if err != nil || replayed.ID != message.ID {
		t.Fatalf("replayed message = %#v, %v", replayed, err)
	}
	if _, err := service.SendMessage(ctx, customerActor, conversation.ID, messageID, "Different"); !errors.Is(err, chatapp.ErrConflict) {
		t.Fatalf("message id conflict = %v", err)
	}

	unread, err := service.UnreadCount(ctx, providerActor)
	if err != nil || unread != 1 {
		t.Fatalf("provider unread = %d, %v", unread, err)
	}
	if err := service.MarkRead(ctx, providerActor, conversation.ID); err != nil {
		t.Fatal(err)
	}
	unread, err = service.UnreadCount(ctx, providerActor)
	if err != nil || unread != 0 {
		t.Fatalf("read provider count = %d, %v", unread, err)
	}

	if err := service.SetActive(ctx, providerActor, conversation.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SendMessage(ctx, customerActor, conversation.ID, uuid.New(), "While open"); err != nil {
		t.Fatal(err)
	}
	unread, err = service.UnreadCount(ctx, providerActor)
	if err != nil || unread != 0 {
		t.Fatalf("active provider unread = %d, %v", unread, err)
	}
	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM chat_push_outbox WHERE recipient_id=$1`, owner).Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("outbox count = %d, %v", outboxCount, err)
	}
	outbox := chatpostgres.NewPushOutbox(pool)
	job, err := outbox.ClaimNext(ctx, time.Now().Add(-5*time.Minute), 8)
	if err != nil || job == nil || job.MessageID != messageID || job.Attempts != 1 || job.RecipientID != owner {
		t.Fatalf("claimed outbox job = %#v, %v", job, err)
	}
	if unavailable, err := outbox.ClaimNext(ctx, time.Now().Add(-5*time.Minute), 8); err != nil || unavailable != nil {
		t.Fatalf("concurrent outbox claim = %#v, %v", unavailable, err)
	}
	retryAt := time.Now().Add(time.Minute)
	if err := outbox.MarkFailed(ctx, messageID, job.Attempts, retryAt, "temporary failure"); err != nil {
		t.Fatal(err)
	}
	if unavailable, err := outbox.ClaimNext(ctx, time.Now().Add(-5*time.Minute), 8); err != nil || unavailable != nil {
		t.Fatalf("early retry claim = %#v, %v", unavailable, err)
	}
	exec(`UPDATE chat_push_outbox SET available_at=now()-interval '1 second' WHERE message_id=$1`, messageID)
	job, err = outbox.ClaimNext(ctx, time.Now().Add(-5*time.Minute), 8)
	if err != nil || job == nil || job.Attempts != 2 {
		t.Fatalf("retried outbox job = %#v, %v", job, err)
	}
	if err := outbox.MarkSent(ctx, messageID, 1); err == nil {
		t.Fatal("stale outbox attempt unexpectedly finalized reclaimed job")
	}
	if err := outbox.MarkSent(ctx, messageID, job.Attempts); err != nil {
		t.Fatal(err)
	}
	var outboxStatus string
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT status::text,attempts FROM chat_push_outbox WHERE message_id=$1`, messageID).Scan(&outboxStatus, &attempts); err != nil || outboxStatus != "sent" || attempts != 2 {
		t.Fatalf("outbox status = %s/%d, %v", outboxStatus, attempts, err)
	}

	if _, err := service.Get(ctx, chatapp.Actor{ID: otherCustomer, Role: "customer"}, conversation.ID); !errors.Is(err, chatapp.ErrNotFound) {
		t.Fatalf("foreign conversation error = %v", err)
	}
	messages, err := service.ListMessages(ctx, customerActor, conversation.ID, nil, 1)
	if err != nil || len(messages.Items) != 1 || messages.NextCursor == nil || messages.Items[0].Text != "While open" {
		t.Fatalf("message page = %#v, %v", messages, err)
	}
}
