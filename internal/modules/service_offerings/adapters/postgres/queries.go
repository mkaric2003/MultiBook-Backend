package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/domain"
)

// Queries is the Postgres read-side adapter for service offerings.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.OfferingQueries = (*Queries)(nil)

const cols = "id,business_id,name,description,duration_minutes,price_minor,is_active,created_at,updated_at"

func (r *Queries) ListOwned(ctx context.Context, actorID string, businessID uuid.UUID) ([]domain.Offering, error) {
	if err := owned(ctx, r.pool, actorID, businessID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, "SELECT "+cols+" FROM service_offerings WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at", businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	offerings := []domain.Offering{}
	for rows.Next() {
		offering, err := scan(rows)
		if err != nil {
			return nil, err
		}
		offerings = append(offerings, offering)
	}
	return offerings, rows.Err()
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func owned(ctx context.Context, db queryRower, actorID string, businessID uuid.UUID) error {
	var valid bool
	err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM businesses WHERE id=$1 AND owner_id=$2 AND type='service' AND deleted_at IS NULL)", businessID, actorID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return application.ErrNotFound
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Offering, error) {
	var offering domain.Offering
	err := row.Scan(&offering.ID, &offering.BusinessID, &offering.Name, &offering.Description, &offering.DurationMinutes, &offering.PriceMinor, &offering.IsActive, &offering.CreatedAt, &offering.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return offering, application.ErrNotFound
	}
	if err != nil {
		return offering, fmt.Errorf("service offering query: %w", err)
	}
	return offering, nil
}
