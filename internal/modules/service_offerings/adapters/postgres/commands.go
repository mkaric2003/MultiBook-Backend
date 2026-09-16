package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for service offerings.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.OfferingRepository = (*CommandRepository)(nil)

func (r *CommandRepository) CreateOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.CreateInput) (domain.Offering, error) {
	var offering domain.Offering
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		created, err := scan(tx.QueryRow(ctx, `INSERT INTO service_offerings(business_id,name,description,duration_minutes,price_minor,is_active) SELECT $1,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM businesses WHERE id=$1 AND owner_id=$2 AND type='service' AND deleted_at IS NULL) RETURNING `+cols, businessID, actorID, input.Name, input.Description, input.DurationMinutes, input.PriceMinor, *input.IsActive))
		offering = created
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_offering_created"})
	})
	return offering, err
}

func (r *CommandRepository) UpdateOwned(ctx context.Context, actorID string, businessID, offeringID uuid.UUID, input domain.UpdateInput) (domain.Offering, error) {
	var offering domain.Offering
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := owned(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		updated, err := scan(tx.QueryRow(ctx, "UPDATE service_offerings SET name=COALESCE($3,name),description=COALESCE($4,description),duration_minutes=COALESCE($5,duration_minutes),price_minor=COALESCE($6,price_minor),is_active=COALESCE($7,is_active) WHERE id=$1 AND business_id=$2 AND deleted_at IS NULL RETURNING "+cols, offeringID, businessID, input.Name, input.Description, input.DurationMinutes, input.PriceMinor, input.IsActive))
		offering = updated
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_offering_updated"})
	})
	return offering, err
}

func (r *CommandRepository) ArchiveOwned(ctx context.Context, actorID string, businessID, offeringID uuid.UUID) error {
	return database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := owned(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, "UPDATE service_offerings SET is_active=FALSE,deleted_at=now() WHERE id=$1 AND business_id=$2 AND deleted_at IS NULL", offeringID, businessID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_offering_archived"})
	})
}
