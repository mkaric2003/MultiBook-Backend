package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.AvailabilityQueries = (*Queries)(nil)

func (r *Queries) ListWeeklyOwned(ctx context.Context, actorID string, businessID, staffID uuid.UUID) ([]domain.WeeklyAvailability, error) {
	if err := requireOwnedStaff(ctx, r.pool, actorID, businessID, staffID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+weeklyColumns+` FROM service_staff_weekly_availability WHERE staff_id = $1 ORDER BY weekday, start_minutes`, staffID)
	if err != nil {
		return nil, fmt.Errorf("list weekly availability: %w", err)
	}
	defer rows.Close()
	items := []domain.WeeklyAvailability{}
	for rows.Next() {
		item, err := scanWeekly(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate weekly availability: %w", err)
	}
	return items, nil
}

func (r *Queries) ListBlocksOwned(ctx context.Context, actorID string, businessID, staffID uuid.UUID) ([]domain.AvailabilityBlock, error) {
	if err := requireOwnedStaff(ctx, r.pool, actorID, businessID, staffID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+blockColumns+` FROM service_staff_availability_blocks WHERE staff_id = $1 ORDER BY lower(blocked_range)`, staffID)
	if err != nil {
		return nil, fmt.Errorf("list availability blocks: %w", err)
	}
	defer rows.Close()
	items := []domain.AvailabilityBlock{}
	for rows.Next() {
		item, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate availability blocks: %w", err)
	}
	return items, nil
}
