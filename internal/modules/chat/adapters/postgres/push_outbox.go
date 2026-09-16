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
)

type PushOutbox struct{ pool *pgxpool.Pool }

func NewPushOutbox(pool *pgxpool.Pool) *PushOutbox { return &PushOutbox{pool: pool} }

var _ application.PushOutbox = (*PushOutbox)(nil)

func (o *PushOutbox) ClaimNext(ctx context.Context, staleBefore time.Time, maximumAttempts int) (*domain.PushJob, error) {
	row := o.pool.QueryRow(ctx, `WITH candidate AS (
		SELECT message_id FROM chat_push_outbox
		WHERE attempts < $1 AND (
			(status IN ('pending','failed') AND available_at <= now())
			OR (status='processing' AND processing_started_at < $2)
		)
		ORDER BY available_at,created_at
		FOR UPDATE SKIP LOCKED LIMIT 1
	), claimed AS (
		UPDATE chat_push_outbox o SET status='processing',attempts=o.attempts+1,
			processing_started_at=now(),last_error=NULL
		FROM candidate WHERE o.message_id=candidate.message_id
		RETURNING o.message_id,o.recipient_id,o.attempts
	)
	SELECT claimed.message_id,claimed.recipient_id,claimed.attempts,
		m.conversation_id,COALESCE(c.business_id::text,c.legacy_business_id),c.business_owner_id,
		c.business_name_snapshot,c.business_image_path_snapshot,c.customer_id,c.customer_name_snapshot,
		c.customer_image_path_snapshot,m.sender_id,m.text
	FROM claimed JOIN chat_messages m ON m.id=claimed.message_id
	JOIN chat_conversations c ON c.id=m.conversation_id`, maximumAttempts, staleBefore)
	var job domain.PushJob
	if err := row.Scan(
		&job.MessageID, &job.RecipientID, &job.Attempts, &job.ConversationID,
		&job.BusinessID, &job.BusinessOwnerID, &job.BusinessName, &job.BusinessImagePath,
		&job.CustomerID, &job.CustomerName, &job.CustomerImagePath, &job.SenderID, &job.MessageText,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim chat push outbox job: %w", err)
	}
	return &job, nil
}

func (o *PushOutbox) MarkSent(ctx context.Context, messageID uuid.UUID, attempt int) error {
	return o.markFinished(ctx, messageID, attempt, "sent")
}

func (o *PushOutbox) MarkSkipped(ctx context.Context, messageID uuid.UUID, attempt int) error {
	return o.markFinished(ctx, messageID, attempt, "skipped")
}

func (o *PushOutbox) markFinished(ctx context.Context, messageID uuid.UUID, attempt int, status string) error {
	result, err := o.pool.Exec(ctx, `UPDATE chat_push_outbox SET status=$2,
		sent_at=now(),processing_started_at=NULL,last_error=NULL
		WHERE message_id=$1 AND status='processing' AND attempts=$3`, messageID, status, attempt)
	if err != nil {
		return fmt.Errorf("mark chat push %s: %w", status, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("mark chat push %s: job is not processing", status)
	}
	return nil
}

func (o *PushOutbox) MarkFailed(ctx context.Context, messageID uuid.UUID, attempt int, retryAt time.Time, failure string) error {
	result, err := o.pool.Exec(ctx, `UPDATE chat_push_outbox SET status='failed',
		available_at=$2,processing_started_at=NULL,last_error=$3
		WHERE message_id=$1 AND status='processing' AND attempts=$4`, messageID, retryAt, failure, attempt)
	if err != nil {
		return fmt.Errorf("mark chat push failed: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("mark chat push failed: job is not processing")
	}
	return nil
}
