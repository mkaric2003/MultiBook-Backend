package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for service details.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.ServiceRepository = (*CommandRepository)(nil)

func (r *CommandRepository) UpsertOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.UpsertInput) (domain.Details, error) {
	var details domain.Details
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO service_details (business_id,time_zone) SELECT $1,$3 WHERE EXISTS(SELECT 1 FROM businesses WHERE id=$1 AND owner_id=$2 AND type='service' AND deleted_at IS NULL) ON CONFLICT(business_id) DO UPDATE SET time_zone=EXCLUDED.time_zone RETURNING business_id,time_zone,created_at,updated_at`, businessID, actorID, input.TimeZone).Scan(&details.BusinessID, &details.TimeZone, &details.CreatedAt, &details.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("upsert service details: %w", err)
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "service_details_updated"})
	})
	return details, err
}
