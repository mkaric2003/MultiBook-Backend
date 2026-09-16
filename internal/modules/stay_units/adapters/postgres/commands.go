// Package postgres implements the stay unit type command persistence port.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for stay unit types.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.UnitTypeRepository = (*CommandRepository)(nil)

func (r *CommandRepository) CreateOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.CreateInput) (domain.UnitType, error) {
	var unitType domain.UnitType
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireMultipleUnitStay(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		created, err := scan(tx.QueryRow(ctx, `
		INSERT INTO stay_unit_types (business_id, name, max_guests, size_square_meters, price_per_night_minor, quantity, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+unitTypeColumns,
			businessID, input.Name, input.MaxGuests, input.SizeSquareMeters, input.PricePerNightMinor, input.Quantity, *input.IsActive))
		unitType = created
		if err != nil {
			return fmt.Errorf("create stay unit type: %w", err)
		}
		if err = insertPhysicalUnits(ctx, tx, businessID, unitType.ID, 1, input.Quantity, *input.IsActive); err != nil {
			return err
		}
		if err = r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "stay_unit_type_created"}); err != nil {
			return fmt.Errorf("audit stay unit type create: %w", err)
		}
		return nil
	})
	return unitType, err
}

func (r *CommandRepository) UpdateOwned(ctx context.Context, actorID string, businessID, unitTypeID uuid.UUID, input domain.UpdateInput) (domain.UnitType, error) {
	var unitType domain.UnitType
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireMultipleUnitStay(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		var previousQuantity int16
		err := tx.QueryRow(ctx, `SELECT quantity FROM stay_unit_types WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL FOR UPDATE`, unitTypeID, businessID).Scan(&previousQuantity)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock stay unit type: %w", err)
		}
		updated, err := scan(tx.QueryRow(ctx, `
		UPDATE stay_unit_types SET name = COALESCE($3, name), max_guests = COALESCE($4, max_guests), size_square_meters = COALESCE($5, size_square_meters), price_per_night_minor = COALESCE($6, price_per_night_minor), quantity = COALESCE($7, quantity), is_active = COALESCE($8, is_active)
		WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL RETURNING `+unitTypeColumns,
			unitTypeID, businessID, input.Name, input.MaxGuests, input.SizeSquareMeters, input.PricePerNightMinor, input.Quantity, input.IsActive))
		unitType = updated
		if err != nil {
			return fmt.Errorf("update stay unit type: %w", err)
		}
		if input.Quantity != nil && *input.Quantity != previousQuantity {
			if err = synchronizePhysicalUnits(ctx, tx, businessID, unitTypeID, previousQuantity, *input.Quantity, unitType.IsActive); err != nil {
				return err
			}
		}
		if input.IsActive != nil {
			if _, err = tx.Exec(ctx, `UPDATE stay_units SET is_active = $3 WHERE business_id = $1 AND stay_unit_type_id = $2 AND deleted_at IS NULL`, businessID, unitTypeID, *input.IsActive); err != nil {
				return fmt.Errorf("update physical unit activity: %w", err)
			}
		}
		if err = r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "stay_unit_type_updated"}); err != nil {
			return fmt.Errorf("audit stay unit type update: %w", err)
		}
		return nil
	})
	return unitType, err
}

func (r *CommandRepository) ArchiveOwned(ctx context.Context, actorID string, businessID, unitTypeID uuid.UUID) error {
	return database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireMultipleUnitStay(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `UPDATE stay_unit_types SET is_active = FALSE, deleted_at = now() WHERE id = $1 AND business_id = $2 AND deleted_at IS NULL`, unitTypeID, businessID)
		if err != nil {
			return fmt.Errorf("archive stay unit type: %w", err)
		}
		if command.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		if _, err = tx.Exec(ctx, `UPDATE stay_units SET is_active = FALSE, deleted_at = now() WHERE business_id = $1 AND stay_unit_type_id = $2 AND deleted_at IS NULL`, businessID, unitTypeID); err != nil {
			return fmt.Errorf("archive physical stay units: %w", err)
		}
		if err = r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "stay_unit_type_archived"}); err != nil {
			return fmt.Errorf("audit stay unit type archive: %w", err)
		}
		return nil
	})
}
