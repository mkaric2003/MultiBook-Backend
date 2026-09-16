package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.NotificationQueries = (*Queries)(nil)

func (r *Queries) List(ctx context.Context, recipientID string, input application.ListInput) (application.Page, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,kind,title,body,data,created_at,read_at FROM in_app_notifications WHERE recipient_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, recipientID, input.Limit+1, input.Offset)
	if err != nil {
		return application.Page{}, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Notification, 0, input.Limit)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return application.Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return application.Page{}, err
	}
	page := application.Page{Items: items}
	if len(items) > input.Limit {
		page.Items = items[:input.Limit]
		nextCursor := strconv.Itoa(input.Offset + input.Limit)
		page.NextCursor = &nextCursor
	}
	return page, nil
}

func (r *Queries) UnreadCount(ctx context.Context, recipientID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM in_app_notifications WHERE recipient_id=$1 AND read_at IS NULL`, recipientID).Scan(&count)
	return count, err
}

func (r *Queries) DeviceTokens(ctx context.Context, recipientID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT token FROM notification_devices WHERE user_id=$1 AND enabled`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := []string{}
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

type notificationScanner interface{ Scan(...any) error }

func scanNotification(row notificationScanner) (domain.Notification, error) {
	var item domain.Notification
	var rawData []byte
	if err := row.Scan(&item.ID, &item.Kind, &item.Title, &item.Body, &rawData, &item.CreatedAt, &item.ReadAt); err != nil {
		return domain.Notification{}, err
	}
	if err := json.Unmarshal(rawData, &item.Data); err != nil {
		return domain.Notification{}, fmt.Errorf("decode notification data: %w", err)
	}
	return item, nil
}
