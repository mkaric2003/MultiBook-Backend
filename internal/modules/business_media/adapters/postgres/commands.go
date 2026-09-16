package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for business media.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.MediaRepository = (*CommandRepository)(nil)

func (r *CommandRepository) ReplaceOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.ReplaceInput) ([]domain.Media, error) {
	var media []domain.Media
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var owned bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM businesses WHERE id=$1 AND owner_id=$2 AND deleted_at IS NULL)", businessID, actorID).Scan(&owned); err != nil {
			return err
		}
		if !owned {
			return application.ErrNotFound
		}
		if _, err := tx.Exec(ctx, "DELETE FROM business_media WHERE business_id=$1", businessID); err != nil {
			return err
		}
		for _, item := range input.Items {
			if _, err := tx.Exec(ctx, "INSERT INTO business_media (business_id,media_type,storage_path,position) VALUES($1,$2,$3,$4)", businessID, item.MediaType, item.StoragePath, item.Position); err != nil {
				return err
			}
		}
		if err := r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "business_media_replaced", Metadata: map[string]any{"count": len(input.Items)}}); err != nil {
			return err
		}
		items, err := list(ctx, tx, businessID)
		media = items
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("replace business media: %w", err)
	}
	return media, nil
}
