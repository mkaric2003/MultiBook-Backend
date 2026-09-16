// Package postgres implements the stays read query port with PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/domain"
)

// Queries is the Postgres read-side adapter for stay details.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.StayQueries = (*Queries)(nil)

func (r *Queries) GetOwned(ctx context.Context, actorID string, businessID uuid.UUID) (domain.Details, error) {
	if err := requireOwnedStay(ctx, r.pool, actorID, businessID); err != nil {
		return domain.Details{}, err
	}
	return get(ctx, r.pool, businessID)
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func requireOwnedStay(ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, actorID string, businessID uuid.UUID) error {
	var owned bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM businesses WHERE id = $1 AND owner_id = $2 AND type = 'stay' AND deleted_at IS NULL)`, businessID, actorID).Scan(&owned)
	if err != nil {
		return fmt.Errorf("check owned stay business: %w", err)
	}
	if !owned {
		return application.ErrNotFound
	}
	return nil
}

func get(ctx context.Context, queryer queryer, businessID uuid.UUID) (domain.Details, error) {
	var details domain.Details
	err := queryer.QueryRow(ctx, `SELECT business_id, inventory_type, base_price_minor, created_at, updated_at FROM stay_details WHERE business_id = $1`, businessID).Scan(&details.BusinessID, &details.InventoryType, &details.BasePriceMinor, &details.CreatedAt, &details.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Details{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Details{}, fmt.Errorf("get stay details: %w", err)
	}
	rows, err := queryer.Query(ctx, "SELECT amenity_code FROM stay_amenities WHERE business_id = $1 ORDER BY amenity_code", businessID)
	if err != nil {
		return domain.Details{}, fmt.Errorf("list stay amenities: %w", err)
	}
	defer rows.Close()
	details.Amenities = []string{}
	for rows.Next() {
		var amenity string
		if err = rows.Scan(&amenity); err != nil {
			return domain.Details{}, fmt.Errorf("scan stay amenity: %w", err)
		}
		details.Amenities = append(details.Amenities, amenity)
	}
	if err = rows.Err(); err != nil {
		return domain.Details{}, fmt.Errorf("iterate stay amenities: %w", err)
	}
	return details, nil
}
