package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/application"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.Repository = (*CommandRepository)(nil)

func (r *CommandRepository) Record(ctx context.Context, customerID string, businessID uuid.UUID) error {
	return database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `INSERT INTO recently_viewed_businesses(customer_id,business_id,viewed_at)
			SELECT $1,$2,now() WHERE EXISTS (SELECT 1 FROM businesses WHERE id=$2 AND status='active' AND deleted_at IS NULL)
			ON CONFLICT(customer_id,business_id) DO UPDATE SET viewed_at=EXCLUDED.viewed_at`, customerID, businessID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return application.ErrBusinessNotFound
		}
		_, err = tx.Exec(ctx, `DELETE FROM recently_viewed_businesses WHERE customer_id=$1 AND business_id IN (
			SELECT business_id FROM recently_viewed_businesses WHERE customer_id=$1
			ORDER BY viewed_at DESC, business_id OFFSET 30)`, customerID)
		return err
	})
}
