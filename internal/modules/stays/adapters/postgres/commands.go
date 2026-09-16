// Package postgres implements the stays command persistence port with PostgreSQL.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for stay details.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.StayRepository = (*CommandRepository)(nil)

func (r *CommandRepository) UpsertOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.UpsertInput) (domain.Details, error) {
	var details domain.Details
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStay(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO stay_details (business_id, inventory_type, base_price_minor) VALUES ($1, $2, $3) ON CONFLICT (business_id) DO UPDATE SET inventory_type = EXCLUDED.inventory_type, base_price_minor = EXCLUDED.base_price_minor`, businessID, input.InventoryType, input.BasePriceMinor); err != nil {
			return fmt.Errorf("upsert stay details: %w", err)
		}
		var err error
		if input.InventoryType == domain.InventoryTypeSingleUnit {
			_, err = tx.Exec(ctx, `INSERT INTO stay_units (business_id, stay_unit_type_id, sequence_number) VALUES ($1, NULL, 1) ON CONFLICT DO NOTHING`, businessID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE stay_units SET is_active = FALSE, deleted_at = now() WHERE business_id = $1 AND stay_unit_type_id IS NULL AND deleted_at IS NULL`, businessID)
		}
		if err != nil {
			return fmt.Errorf("synchronize stay physical unit: %w", err)
		}
		if _, err = tx.Exec(ctx, "DELETE FROM stay_amenities WHERE business_id = $1", businessID); err != nil {
			return fmt.Errorf("clear stay amenities: %w", err)
		}
		for _, amenity := range input.Amenities {
			if _, err = tx.Exec(ctx, "INSERT INTO stay_amenities (business_id, amenity_code) VALUES ($1, $2)", businessID, amenity); err != nil {
				return fmt.Errorf("insert stay amenity: %w", err)
			}
		}
		if err = r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "stay_details_updated"}); err != nil {
			return fmt.Errorf("audit stay update: %w", err)
		}
		updated, err := get(ctx, tx, businessID)
		details = updated
		if err != nil {
			return err
		}
		return nil
	})
	return details, err
}
