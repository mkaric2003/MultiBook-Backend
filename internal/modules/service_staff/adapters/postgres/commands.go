package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for service staff.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.StaffRepository = (*CommandRepository)(nil)

func (r *CommandRepository) CreateOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.CreateInput) (domain.Staff, error) {
	var staff domain.Staff
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := owned(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		created, err := scan(tx.QueryRow(ctx, "INSERT INTO service_staff(business_id,name,title,commission_rate,is_active)VALUES($1,$2,$3,$4,$5)RETURNING "+cols, businessID, input.Name, input.Title, *input.CommissionRate, *input.IsActive))
		staff = created
		if err != nil {
			return err
		}
		if err = links(ctx, tx, businessID, staff.ID, input.OfferingIDs); err != nil {
			return err
		}
		staff.OfferingIDs = append([]uuid.UUID{}, input.OfferingIDs...)
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_created"})
	})
	return staff, err
}

func (r *CommandRepository) UpdateOwned(ctx context.Context, actorID string, businessID, staffID uuid.UUID, input domain.UpdateInput) (domain.Staff, error) {
	var staff domain.Staff
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := owned(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		updated, err := scan(tx.QueryRow(ctx, "UPDATE service_staff SET name=COALESCE($3,name),title=COALESCE($4,title),commission_rate=COALESCE($5,commission_rate),is_active=COALESCE($6,is_active)WHERE id=$1 AND business_id=$2 AND deleted_at IS NULL RETURNING "+cols, staffID, businessID, input.Name, input.Title, input.CommissionRate, input.IsActive))
		staff = updated
		if err != nil {
			return err
		}
		if input.OfferingIDs != nil {
			if _, err = tx.Exec(ctx, "DELETE FROM service_staff_offerings WHERE staff_id=$1", staffID); err != nil {
				return err
			}
			if err = links(ctx, tx, businessID, staffID, *input.OfferingIDs); err != nil {
				return err
			}
			staff.OfferingIDs = append([]uuid.UUID{}, (*input.OfferingIDs)...)
			return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_updated"})
		}
		staff.OfferingIDs, err = offerings(ctx, tx, staffID)
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_updated"})
	})
	return staff, err
}

func (r *CommandRepository) ArchiveOwned(ctx context.Context, actorID string, businessID, staffID uuid.UUID) error {
	return database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := owned(ctx, tx, actorID, businessID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, "UPDATE service_staff SET is_active=FALSE,deleted_at=now() WHERE id=$1 AND business_id=$2 AND deleted_at IS NULL", staffID, businessID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_archived"})
	})
}
