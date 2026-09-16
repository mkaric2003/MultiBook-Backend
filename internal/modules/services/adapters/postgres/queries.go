package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/domain"
)

// Queries is the Postgres read-side adapter for service details.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.ServiceQueries = (*Queries)(nil)

func (r *Queries) GetOwned(ctx context.Context, actorID string, businessID uuid.UUID) (domain.Details, error) {
	var details domain.Details
	err := r.pool.QueryRow(ctx, `SELECT d.business_id,d.time_zone,d.created_at,d.updated_at FROM service_details d JOIN businesses b ON b.id=d.business_id WHERE d.business_id=$1 AND b.owner_id=$2 AND b.type='service' AND b.deleted_at IS NULL`, businessID, actorID).Scan(&details.BusinessID, &details.TimeZone, &details.CreatedAt, &details.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return details, application.ErrNotFound
	}
	return details, err
}
