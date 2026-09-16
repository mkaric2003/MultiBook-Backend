package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/domain"
)

// Queries is the Postgres read-side adapter for service staff.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.StaffQueries = (*Queries)(nil)

const cols = "id,business_id,name,title,commission_rate,is_active,created_at,updated_at"

func (r *Queries) ListOwned(ctx context.Context, actorID string, businessID uuid.UUID) ([]domain.Staff, error) {
	if err := owned(ctx, r.pool, actorID, businessID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, "SELECT "+cols+" FROM service_staff WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at", businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	staffMembers := []domain.Staff{}
	for rows.Next() {
		staff, err := scan(rows)
		if err != nil {
			return nil, err
		}
		staff.OfferingIDs, _ = offerings(ctx, r.pool, staff.ID)
		staffMembers = append(staffMembers, staff)
	}
	return staffMembers, rows.Err()
}

type db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func owned(ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, actorID string, businessID uuid.UUID) error {
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

func links(ctx context.Context, db db, businessID, staffID uuid.UUID, offeringIDs []uuid.UUID) error {
	for _, offeringID := range offeringIDs {
		command, err := db.Exec(ctx, "INSERT INTO service_staff_offerings(staff_id,offering_id) SELECT $1,$2 WHERE EXISTS(SELECT 1 FROM service_offerings WHERE id=$2 AND business_id=$3 AND deleted_at IS NULL)", staffID, offeringID, businessID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return application.ErrValidation
		}
	}
	return nil
}

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func offerings(ctx context.Context, db querier, staffID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, "SELECT offering_id FROM service_staff_offerings WHERE staff_id=$1", staffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	offeringIDs := []uuid.UUID{}
	for rows.Next() {
		var offeringID uuid.UUID
		if err = rows.Scan(&offeringID); err != nil {
			return nil, err
		}
		offeringIDs = append(offeringIDs, offeringID)
	}
	return offeringIDs, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Staff, error) {
	var staff domain.Staff
	err := row.Scan(&staff.ID, &staff.BusinessID, &staff.Name, &staff.Title, &staff.CommissionRate, &staff.IsActive, &staff.CreatedAt, &staff.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return staff, application.ErrNotFound
	}
	if err != nil {
		return staff, fmt.Errorf("staff query: %w", err)
	}
	return staff, nil
}
