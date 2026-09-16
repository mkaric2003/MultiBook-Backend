package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.NotificationCommands = (*CommandRepository)(nil)

func (r *CommandRepository) Create(ctx context.Context, event domain.Event) error {
	rawData, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("encode notification data: %w", err)
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO in_app_notifications(id,recipient_id,kind,title,body,data) VALUES($1,$2,$3,$4,$5,$6::jsonb) ON CONFLICT (id) DO NOTHING`, event.ID, event.RecipientID, event.Kind, event.Title, event.Body, rawData)
	return err
}

func (r *CommandRepository) MarkRead(ctx context.Context, recipientID, notificationID string) error {
	result, err := r.pool.Exec(ctx, `UPDATE in_app_notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1 AND recipient_id=$2`, notificationID, recipientID)
	if err != nil {
		return fmt.Errorf("mark notification as read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *CommandRepository) UpsertDevice(ctx context.Context, recipientID, deviceID, token string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO notification_devices(user_id,device_id,token,enabled) VALUES($1,$2,$3,TRUE) ON CONFLICT(user_id,device_id) DO UPDATE SET token=EXCLUDED.token,enabled=TRUE,updated_at=now()`, recipientID, deviceID, token)
	return err
}

func (r *CommandRepository) DeleteDevice(ctx context.Context, recipientID, deviceID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_devices WHERE user_id=$1 AND device_id=$2`, recipientID, deviceID)
	return err
}
