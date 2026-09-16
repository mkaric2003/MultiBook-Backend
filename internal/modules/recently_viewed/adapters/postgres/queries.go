package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/application"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (q *Queries) ListBusinessIDs(ctx context.Context, customerID string, businessType domain.BusinessType, limit int) ([]uuid.UUID, error) {
	rows, err := q.pool.Query(ctx, `SELECT rv.business_id FROM recently_viewed_businesses rv
		JOIN businesses b ON b.id=rv.business_id
		WHERE rv.customer_id=$1 AND b.type=$2 AND b.status='active' AND b.deleted_at IS NULL
		ORDER BY rv.viewed_at DESC, rv.business_id LIMIT $3`, customerID, businessType, limit)
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
