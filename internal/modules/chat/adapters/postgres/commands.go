package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.Commands = (*CommandRepository)(nil)

func (r *CommandRepository) GetOrCreate(ctx context.Context, actor application.Actor, businessID uuid.UUID, customerID string) (domain.Conversation, error) {
	var conversation domain.Conversation
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		conversation, err = getByBusinessAndCustomer(ctx, tx, businessID, customerID)
		if err == nil {
			if actor.ID != conversation.CustomerID && actor.ID != conversation.BusinessOwnerID {
				return application.ErrForbidden
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find chat conversation: %w", err)
		}

		var ownerID, businessName string
		var businessImagePath *string
		err = tx.QueryRow(ctx, `SELECT b.owner_id,b.name,
			(SELECT bm.storage_path FROM business_media bm WHERE bm.business_id=b.id
			 AND bm.media_type IN ('logo','cover') ORDER BY CASE bm.media_type WHEN 'logo' THEN 0 ELSE 1 END LIMIT 1)
			FROM businesses b WHERE b.id=$1 AND b.deleted_at IS NULL AND b.status='active'`, businessID).
			Scan(&ownerID, &businessName, &businessImagePath)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load chat business: %w", err)
		}
		if actor.Role == "provider" && ownerID != actor.ID {
			return application.ErrForbidden
		}
		if ownerID == customerID {
			return application.ErrValidation
		}

		var customerName string
		var customerImagePath *string
		err = tx.QueryRow(ctx, `SELECT COALESCE(
			NULLIF(btrim(concat_ws(' ',first_name,last_name)),''),NULLIF(btrim(display_name),''),
			NULLIF(btrim(email::text),''),'Customer'),avatar_storage_path
			FROM users WHERE id=$1 AND deleted_at IS NULL AND role='customer'`, customerID).
			Scan(&customerName, &customerImagePath)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load chat customer: %w", err)
		}

		conversationID := uuid.New()
		err = tx.QueryRow(ctx, `INSERT INTO chat_conversations(
			id,business_id,business_owner_id,customer_id,business_name_snapshot,business_image_path_snapshot,
			customer_name_snapshot,customer_image_path_snapshot)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (business_id,customer_id) WHERE business_id IS NOT NULL
			DO UPDATE SET business_name_snapshot=EXCLUDED.business_name_snapshot
			RETURNING id`, conversationID, businessID, ownerID, customerID, businessName,
			businessImagePath, customerName, customerImagePath).Scan(&conversationID)
		if err != nil {
			return fmt.Errorf("create chat conversation: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO chat_participant_state(conversation_id,user_id,participant_role)
			VALUES($1,$2,'customer'),($1,$3,'business') ON CONFLICT DO NOTHING`, conversationID, customerID, ownerID); err != nil {
			return fmt.Errorf("create chat participant state: %w", err)
		}
		conversation, err = getByBusinessAndCustomer(ctx, tx, businessID, customerID)
		return err
	})
	return conversation, err
}

func getByBusinessAndCustomer(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, customerID string) (domain.Conversation, error) {
	return scanConversation(tx.QueryRow(ctx, `SELECT `+conversationColumns+conversationJoins+`
		WHERE c.business_id=$1 AND c.customer_id=$2 FOR UPDATE OF c`, businessID, customerID))
}

func (r *CommandRepository) SendMessage(ctx context.Context, actorID string, conversationID, messageID uuid.UUID, text string) (domain.Message, error) {
	var message domain.Message
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var recipientID string
		var recipientActive bool
		err := tx.QueryRow(ctx, `SELECT recipient.user_id,COALESCE(recipient.active_until > now(),FALSE)
			FROM chat_participant_state sender
			JOIN chat_participant_state recipient ON recipient.conversation_id=sender.conversation_id AND recipient.user_id<>sender.user_id
			WHERE sender.conversation_id=$1 AND sender.user_id=$2 FOR UPDATE OF sender,recipient`, conversationID, actorID).
			Scan(&recipientID, &recipientActive)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("authorize chat sender: %w", err)
		}

		err = tx.QueryRow(ctx, `SELECT id,conversation_id,sender_id,text,created_at FROM chat_messages WHERE id=$1`, messageID).
			Scan(&message.ID, &message.ConversationID, &message.SenderID, &message.Text, &message.CreatedAt)
		if err == nil {
			if message.ConversationID != conversationID || message.SenderID != actorID || message.Text != text {
				return application.ErrConflict
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("check chat message id: %w", err)
		}

		message = domain.Message{ID: messageID, ConversationID: conversationID, SenderID: actorID, Text: text}
		if err := tx.QueryRow(ctx, `INSERT INTO chat_messages(id,conversation_id,sender_id,text)
			VALUES($1,$2,$3,$4) RETURNING created_at`, message.ID, message.ConversationID, message.SenderID, message.Text).
			Scan(&message.CreatedAt); err != nil {
			return fmt.Errorf("insert chat message: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE chat_conversations
			SET last_message_id=$2,last_message_text=$3,last_message_at=$4 WHERE id=$1`,
			conversationID, messageID, text, message.CreatedAt); err != nil {
			return fmt.Errorf("update chat conversation preview: %w", err)
		}
		if recipientActive {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE chat_participant_state SET unread_count=unread_count+1
			WHERE conversation_id=$1 AND user_id=$2`, conversationID, recipientID); err != nil {
			return fmt.Errorf("increment chat unread count: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO chat_push_outbox(message_id,recipient_id) VALUES($1,$2)`, messageID, recipientID); err != nil {
			return fmt.Errorf("enqueue chat push: %w", err)
		}
		return nil
	})
	return message, err
}

func (r *CommandRepository) MarkRead(ctx context.Context, actorID string, conversationID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `UPDATE chat_participant_state state SET
		unread_count=0,last_read_message_id=c.last_message_id,
		last_read_at=CASE WHEN c.last_message_id IS NULL THEN NULL ELSE now() END
		FROM chat_conversations c WHERE c.id=$1 AND state.conversation_id=c.id AND state.user_id=$2`, conversationID, actorID)
	if err != nil {
		return fmt.Errorf("mark chat conversation read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *CommandRepository) SetTyping(ctx context.Context, actorID string, conversationID uuid.UUID, until *time.Time) error {
	return r.updateExpiry(ctx, actorID, conversationID, "typing_until", until)
}

func (r *CommandRepository) SetActive(ctx context.Context, actorID string, conversationID uuid.UUID, until *time.Time) error {
	return r.updateExpiry(ctx, actorID, conversationID, "active_until", until)
}

func (r *CommandRepository) updateExpiry(ctx context.Context, actorID string, conversationID uuid.UUID, column string, until *time.Time) error {
	result, err := r.pool.Exec(ctx, `UPDATE chat_participant_state SET `+column+`=$3 WHERE conversation_id=$1 AND user_id=$2`, conversationID, actorID, until)
	if err != nil {
		return fmt.Errorf("update chat participant expiry: %w", err)
	}
	if result.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}
