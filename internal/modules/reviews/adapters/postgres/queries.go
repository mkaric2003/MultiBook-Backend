package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (q *Queries) HasReview(ctx context.Context, customerID string, businessID uuid.UUID) (bool, error) {
	var exists bool
	err := q.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM business_reviews WHERE business_id=$1 AND customer_id=$2)`, businessID, customerID).Scan(&exists)
	return exists, err
}

func (q *Queries) List(ctx context.Context, businessID uuid.UUID, limit, offset int) ([]domain.Review, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,business_id,business_owner_id,customer_id,customer_name,
		customer_avatar_path,COALESCE(stay_booking_id,service_appointment_id),
		CASE WHEN stay_booking_id IS NOT NULL THEN 'stay' ELSE 'service' END,
		rating,comment,created_at
		FROM business_reviews WHERE business_id=$1
		ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, businessID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Review, 0)
	for rows.Next() {
		var item domain.Review
		if err := rows.Scan(&item.ID, &item.BusinessID, &item.BusinessOwnerID, &item.CustomerID, &item.CustomerName,
			&item.CustomerAvatarPath, &item.SourceID, &item.SourceType, &item.Rating, &item.Comment, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
