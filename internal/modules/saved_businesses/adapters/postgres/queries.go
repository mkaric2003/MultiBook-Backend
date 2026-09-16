package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/application"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (q *Queries) IsSaved(ctx context.Context, customerID string, businessID uuid.UUID) (bool, error) {
	var saved bool
	err := q.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM saved_businesses s JOIN businesses b ON b.id=s.business_id
		WHERE s.customer_id=$1 AND s.business_id=$2 AND b.status='active' AND b.deleted_at IS NULL
	)`, customerID, businessID).Scan(&saved)
	return saved, err
}

func (q *Queries) ListBusinessIDs(ctx context.Context, customerID string) ([]uuid.UUID, error) {
	rows, err := q.pool.Query(ctx, `SELECT business_id FROM saved_businesses
		WHERE customer_id=$1 ORDER BY saved_at DESC, business_id`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
