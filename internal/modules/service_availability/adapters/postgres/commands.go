// Package postgres implements service staff availability persistence.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.AvailabilityRepository = (*CommandRepository)(nil)

func (r *CommandRepository) CreateWeeklyOwned(ctx context.Context, actorID string, businessID, staffID uuid.UUID, input domain.CreateWeeklyInput) (domain.WeeklyAvailability, error) {
	var availability domain.WeeklyAvailability
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStaff(ctx, tx, actorID, businessID, staffID); err != nil {
			return err
		}
		created, err := scanWeekly(tx.QueryRow(ctx, `INSERT INTO service_staff_weekly_availability (staff_id, weekday, start_minutes, end_minutes) VALUES ($1, $2, $3, $4) RETURNING `+weeklyColumns, staffID, input.Weekday, input.StartMinutes, input.EndMinutes))
		availability = created
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_weekly_availability_created"})
	})
	return availability, availabilityError(err)
}
func (r *CommandRepository) UpdateWeeklyOwned(ctx context.Context, actorID string, businessID, staffID, availabilityID uuid.UUID, input domain.UpdateWeeklyInput) (domain.WeeklyAvailability, error) {
	var availability domain.WeeklyAvailability
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStaff(ctx, tx, actorID, businessID, staffID); err != nil {
			return err
		}
		updated, err := scanWeekly(tx.QueryRow(ctx, `UPDATE service_staff_weekly_availability SET weekday = COALESCE($3, weekday), start_minutes = COALESCE($4, start_minutes), end_minutes = COALESCE($5, end_minutes) WHERE id = $1 AND staff_id = $2 RETURNING `+weeklyColumns, availabilityID, staffID, input.Weekday, input.StartMinutes, input.EndMinutes))
		availability = updated
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_weekly_availability_updated"})
	})
	return availability, availabilityError(err)
}
func (r *CommandRepository) DeleteWeeklyOwned(ctx context.Context, actorID string, businessID, staffID, availabilityID uuid.UUID) error {
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStaff(ctx, tx, actorID, businessID, staffID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `DELETE FROM service_staff_weekly_availability WHERE id = $1 AND staff_id = $2`, availabilityID, staffID)
		if err != nil {
			return fmt.Errorf("delete weekly availability: %w", err)
		}
		if command.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_weekly_availability_deleted"})
	})
	return availabilityError(err)
}

func (r *CommandRepository) CreateBlockOwned(ctx context.Context, actorID string, businessID, staffID uuid.UUID, input domain.CreateBlockInput) (domain.AvailabilityBlock, error) {
	var block domain.AvailabilityBlock
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStaff(ctx, tx, actorID, businessID, staffID); err != nil {
			return err
		}
		created, err := scanBlock(tx.QueryRow(ctx, `INSERT INTO service_staff_availability_blocks (staff_id, blocked_range, reason) VALUES ($1, tstzrange($2, $3, '[)'), $4) RETURNING `+blockColumns, staffID, input.StartAt, input.EndAt, input.Reason))
		block = created
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_availability_block_created"})
	})
	return block, availabilityError(err)
}
func (r *CommandRepository) UpdateBlockOwned(ctx context.Context, actorID string, businessID, staffID, blockID uuid.UUID, input domain.UpdateBlockInput) (domain.AvailabilityBlock, error) {
	var block domain.AvailabilityBlock
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStaff(ctx, tx, actorID, businessID, staffID); err != nil {
			return err
		}
		updated, err := scanBlock(tx.QueryRow(ctx, `UPDATE service_staff_availability_blocks SET blocked_range = tstzrange(COALESCE($3, lower(blocked_range)), COALESCE($4, upper(blocked_range)), '[)'), reason = COALESCE($5, reason) WHERE id = $1 AND staff_id = $2 RETURNING `+blockColumns, blockID, staffID, input.StartAt, input.EndAt, input.Reason))
		block = updated
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_availability_block_updated"})
	})
	return block, availabilityError(err)
}
func (r *CommandRepository) DeleteBlockOwned(ctx context.Context, actorID string, businessID, staffID, blockID uuid.UUID) error {
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		if err := requireOwnedStaff(ctx, tx, actorID, businessID, staffID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `DELETE FROM service_staff_availability_blocks WHERE id = $1 AND staff_id = $2`, blockID, staffID)
		if err != nil {
			return fmt.Errorf("delete availability block: %w", err)
		}
		if command.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_staff_availability_block_deleted"})
	})
	return availabilityError(err)
}

func availabilityError(err error) error {
	if err == nil || errors.Is(err, application.ErrNotFound) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23P01" || pgErr.Code == "23514") {
		return fmt.Errorf("%w: availability interval conflicts with an existing interval", application.ErrValidation)
	}
	return err
}
