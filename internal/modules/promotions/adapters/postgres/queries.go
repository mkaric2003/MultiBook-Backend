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
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

const promotionColumns = `id,business_id,owner_id,name,type::text,value,starts_at,ends_at,is_active,code::text,minimum_amount_minor,minimum_nights,usage_limit,usage_count,created_at`

func (queries *Queries) ListOwned(ctx context.Context, ownerID string, businessID uuid.UUID) ([]domain.Promotion, error) {
	rows, err := queries.pool.Query(ctx, `SELECT `+promotionColumns+` FROM business_promotions WHERE business_id=$1 AND owner_id=$2 ORDER BY created_at DESC,id DESC`, businessID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list promotions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Promotion, 0)
	for rows.Next() {
		promotion, err := scanPromotion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, promotion)
	}
	return items, rows.Err()
}

func (queries *Queries) Active(ctx context.Context, businessID uuid.UUID, promoCode *string) (*domain.Promotion, error) {
	promotion, err := activePromotion(ctx, queries.pool, businessID, promoCode, "")
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active promotion: %w", err)
	}
	return &promotion, nil
}

type rowScanner interface{ Scan(...any) error }

func scanPromotion(row rowScanner) (domain.Promotion, error) {
	var promotion domain.Promotion
	var promotionType string
	err := row.Scan(&promotion.ID, &promotion.BusinessID, &promotion.OwnerID, &promotion.Name, &promotionType, &promotion.Value, &promotion.StartsAt, &promotion.EndsAt, &promotion.IsActive, &promotion.Code, &promotion.MinimumAmount, &promotion.MinimumNights, &promotion.UsageLimit, &promotion.UsageCount, &promotion.CreatedAt)
	promotion.Type = domain.Type(promotionType)
	return promotion, err
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func activePromotion(ctx context.Context, source queryRower, businessID uuid.UUID, promoCode *string, suffix string) (domain.Promotion, error) {
	where := `type <> 'couponCode'`
	args := []any{businessID}
	if promoCode != nil {
		where = `type = 'couponCode' AND code = $2`
		args = append(args, *promoCode)
	}
	query := `SELECT ` + promotionColumns + ` FROM business_promotions WHERE business_id=$1 AND is_active AND starts_at <= now() AND ends_at >= now() AND (usage_limit IS NULL OR usage_count < usage_limit) AND ` + where + ` ORDER BY value DESC,created_at DESC,id DESC LIMIT 1 ` + suffix
	return scanPromotion(source.QueryRow(ctx, query, args...))
}
