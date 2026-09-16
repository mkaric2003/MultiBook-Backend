package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/application"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.Repository = (*CommandRepository)(nil)

func (r *CommandRepository) Save(ctx context.Context, customerID string, businessID uuid.UUID) error {
	command, err := r.pool.Exec(ctx, `INSERT INTO saved_businesses(customer_id,business_id)
		SELECT $1,$2 WHERE EXISTS (SELECT 1 FROM businesses WHERE id=$2 AND status='active' AND deleted_at IS NULL)
		ON CONFLICT(customer_id,business_id) DO UPDATE SET business_id=EXCLUDED.business_id`, customerID, businessID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return application.ErrBusinessNotFound
	}
	return nil
}

func (r *CommandRepository) Remove(ctx context.Context, customerID string, businessID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM saved_businesses WHERE customer_id=$1 AND business_id=$2`, customerID, businessID)
	return err
}
