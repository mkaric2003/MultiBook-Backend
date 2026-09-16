package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.CommandRepository = (*CommandRepository)(nil)

func (repository *CommandRepository) Create(ctx context.Context, ownerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Promotion, error) {
	var promotion domain.Promotion
	err := database.WithinTransaction(ctx, repository.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `INSERT INTO business_promotions(business_id,owner_id,name,type,value,starts_at,ends_at,code,minimum_amount_minor,minimum_nights,usage_limit)
			SELECT b.id,b.owner_id,$3,$4::promotion_type,$5,$6,$7,$8,$9,$10,$11 FROM businesses b
			WHERE b.id=$1 AND b.owner_id=$2 AND b.deleted_at IS NULL
			RETURNING `+promotionColumns, businessID, ownerID, input.Name, input.Type, input.Value, input.StartsAt, input.EndsAt, input.Code, input.MinimumAmount, input.MinimumNights, input.UsageLimit)
		var err error
		promotion, err = scanPromotion(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("create promotion: %w", err)
		}
		return refreshBusinessPromotion(ctx, tx, businessID)
	})
	return promotion, err
}

func (repository *CommandRepository) SetActive(ctx context.Context, ownerID string, businessID, promotionID uuid.UUID, active bool) (domain.Promotion, error) {
	var promotion domain.Promotion
	err := database.WithinTransaction(ctx, repository.pool, func(tx pgx.Tx) error {
		var err error
		promotion, err = scanPromotion(tx.QueryRow(ctx, `UPDATE business_promotions SET is_active=$4 WHERE id=$1 AND business_id=$2 AND owner_id=$3 RETURNING `+promotionColumns, promotionID, businessID, ownerID, active))
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("update promotion: %w", err)
		}
		return refreshBusinessPromotion(ctx, tx, businessID)
	})
	return promotion, err
}

func (repository *CommandRepository) Delete(ctx context.Context, ownerID string, businessID, promotionID uuid.UUID) error {
	return database.WithinTransaction(ctx, repository.pool, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `DELETE FROM business_promotions WHERE id=$1 AND business_id=$2 AND owner_id=$3`, promotionID, businessID, ownerID)
		if err != nil {
			return fmt.Errorf("delete promotion: %w", err)
		}
		if command.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		return refreshBusinessPromotion(ctx, tx, businessID)
	})
}

func refreshBusinessPromotion(ctx context.Context, tx pgx.Tx, businessID uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE businesses SET is_promotion_active=EXISTS(
		SELECT 1 FROM business_promotions WHERE business_id=$1 AND is_active AND starts_at <= now() AND ends_at >= now() AND (usage_limit IS NULL OR usage_count < usage_limit)
	) WHERE id=$1`, businessID)
	if err != nil {
		return fmt.Errorf("refresh business promotion status: %w", err)
	}
	return nil
}
