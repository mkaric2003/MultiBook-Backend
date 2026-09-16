package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/domain"
)

const weeklyColumns = "id, staff_id, weekday, start_minutes, end_minutes, created_at"
const blockColumns = "id, staff_id, lower(blocked_range), upper(blocked_range), reason, created_at"

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func requireOwnedStaff(ctx context.Context, db queryer, actorID string, businessID, staffID uuid.UUID) error {
	var owned bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM businesses b JOIN service_staff s ON s.business_id = b.id WHERE b.id = $1 AND b.owner_id = $2 AND b.type = 'service' AND b.deleted_at IS NULL AND s.id = $3 AND s.deleted_at IS NULL AND s.is_active)`, businessID, actorID, staffID).Scan(&owned)
	if err != nil {
		return fmt.Errorf("check owned service staff: %w", err)
	}
	if !owned {
		return application.ErrNotFound
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanWeekly(row scanner) (domain.WeeklyAvailability, error) {
	var item domain.WeeklyAvailability
	err := row.Scan(&item.ID, &item.StaffID, &item.Weekday, &item.StartMinutes, &item.EndMinutes, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, application.ErrNotFound
	}
	return item, err
}

func scanBlock(row scanner) (domain.AvailabilityBlock, error) {
	var item domain.AvailabilityBlock
	err := row.Scan(&item.ID, &item.StaffID, &item.StartAt, &item.EndAt, &item.Reason, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, application.ErrNotFound
	}
	return item, err
}
