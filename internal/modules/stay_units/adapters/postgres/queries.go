// Package postgres implements the stay unit type read query port.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/domain"
)

// Queries is the Postgres read-side adapter for stay unit types.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.UnitTypeQueries = (*Queries)(nil)

const unitTypeColumns = `id, business_id, name, max_guests, size_square_meters,
	price_per_night_minor, quantity, is_active, created_at, updated_at`

func (r *Queries) ListOwned(ctx context.Context, actorID string, businessID uuid.UUID) ([]domain.UnitType, error) {
	if err := requireMultipleUnitStay(ctx, r.pool, actorID, businessID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+unitTypeColumns+` FROM stay_unit_types WHERE business_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list stay unit types: %w", err)
	}
	defer rows.Close()
	unitTypes := []domain.UnitType{}
	for rows.Next() {
		unitType, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stay unit type: %w", err)
		}
		unitTypes = append(unitTypes, unitType)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stay unit types: %w", err)
	}
	return unitTypes, nil
}

type db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func requireMultipleUnitStay(ctx context.Context, db db, actorID string, businessID uuid.UUID) error {
	var valid bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM businesses b JOIN stay_details d ON d.business_id = b.id WHERE b.id = $1 AND b.owner_id = $2 AND b.type = 'stay' AND b.deleted_at IS NULL AND d.inventory_type = 'multiple_units')`, businessID, actorID).Scan(&valid)
	if err != nil {
		return fmt.Errorf("check owned multiple-unit stay: %w", err)
	}
	if !valid {
		return application.ErrNotFound
	}
	return nil
}

func insertPhysicalUnits(ctx context.Context, db db, businessID, unitTypeID uuid.UUID, from, to int16, active bool) error {
	for sequence := from; sequence <= to; sequence++ {
		if _, err := db.Exec(ctx, `INSERT INTO stay_units (business_id, stay_unit_type_id, sequence_number, is_active) VALUES ($1, $2, $3, $4)`, businessID, unitTypeID, sequence, active); err != nil {
			return fmt.Errorf("create physical stay unit: %w", err)
		}
	}
	return nil
}

func synchronizePhysicalUnits(ctx context.Context, db db, businessID, unitTypeID uuid.UUID, previous, next int16, active bool) error {
	if next > previous {
		return insertPhysicalUnits(ctx, db, businessID, unitTypeID, previous+1, next, active)
	}
	if next < previous {
		if _, err := db.Exec(ctx, `UPDATE stay_units SET is_active = FALSE, deleted_at = now() WHERE business_id = $1 AND stay_unit_type_id = $2 AND sequence_number > $3 AND deleted_at IS NULL`, businessID, unitTypeID, next); err != nil {
			return fmt.Errorf("archive surplus physical stay units: %w", err)
		}
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.UnitType, error) {
	var unitType domain.UnitType
	err := row.Scan(&unitType.ID, &unitType.BusinessID, &unitType.Name, &unitType.MaxGuests, &unitType.SizeSquareMeters, &unitType.PricePerNightMinor, &unitType.Quantity, &unitType.IsActive, &unitType.CreatedAt, &unitType.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UnitType{}, application.ErrNotFound
	}
	return unitType, err
}
