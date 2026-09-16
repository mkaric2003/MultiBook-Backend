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

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (q *Queries) Get(ctx context.Context, actorID string, conversationID uuid.UUID) (domain.Conversation, error) {
	item, err := scanConversation(q.pool.QueryRow(ctx, `SELECT `+conversationColumns+conversationJoins+`
		JOIN chat_participant_state actor_state ON actor_state.conversation_id=c.id AND actor_state.user_id=$2
		WHERE c.id=$1`, conversationID, actorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Conversation{}, application.ErrNotFound
	}
	return item, err
}

func (q *Queries) List(ctx context.Context, actorID string, cursor *application.Cursor, limit int) ([]domain.Conversation, error) {
	var cursorAt *time.Time
	cursorID := uuid.Nil
	if cursor != nil {
		cursorAt, cursorID = &cursor.At, cursor.ID
	}
	rows, err := q.pool.Query(ctx, `SELECT `+conversationColumns+conversationJoins+`
		JOIN chat_participant_state actor_state ON actor_state.conversation_id=c.id AND actor_state.user_id=$1
		WHERE ($2::timestamptz IS NULL OR (COALESCE(c.last_message_at,c.created_at),c.id) < ($2,$3))
		ORDER BY COALESCE(c.last_message_at,c.created_at) DESC,c.id DESC LIMIT $4`, actorID, cursorAt, cursorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list chat conversations: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Conversation, 0, limit)
	for rows.Next() {
		item, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) ListMessages(ctx context.Context, actorID string, conversationID uuid.UUID, cursor *application.Cursor, limit int) ([]domain.Message, error) {
	var participant bool
	if err := q.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_participant_state WHERE conversation_id=$1 AND user_id=$2)`, conversationID, actorID).Scan(&participant); err != nil {
		return nil, err
	}
	if !participant {
		return nil, application.ErrNotFound
	}
	var cursorAt *time.Time
	cursorID := uuid.Nil
	if cursor != nil {
		cursorAt, cursorID = &cursor.At, cursor.ID
	}
	rows, err := q.pool.Query(ctx, `SELECT id,conversation_id,sender_id,text,created_at FROM chat_messages
		WHERE conversation_id=$1 AND ($2::timestamptz IS NULL OR (created_at,id)<($2,$3))
		ORDER BY created_at DESC,id DESC LIMIT $4`, conversationID, cursorAt, cursorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Message, 0, limit)
	for rows.Next() {
		var item domain.Message
		if err := rows.Scan(&item.ID, &item.ConversationID, &item.SenderID, &item.Text, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) UnreadCount(ctx context.Context, actorID string) (int, error) {
	var count int
	err := q.pool.QueryRow(ctx, `SELECT COALESCE(sum(unread_count),0) FROM chat_participant_state WHERE user_id=$1`, actorID).Scan(&count)
	return count, err
}
