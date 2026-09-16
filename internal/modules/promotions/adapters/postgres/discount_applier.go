package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
)

type DiscountApplier struct{}

func NewDiscountApplier() *DiscountApplier { return &DiscountApplier{} }

var _ application.DiscountApplier = (*DiscountApplier)(nil)

func (applier *DiscountApplier) Apply(ctx context.Context, tx pgx.Tx, input domain.DiscountInput) (*domain.AppliedDiscount, error) {
	code := input.PromoCode
	if code != nil {
		normalized := strings.TrimSpace(*code)
		if normalized == "" {
			code = nil
		} else {
			code = &normalized
		}
	}
	promotion, err := activePromotion(ctx, tx, input.BusinessID, code, "FOR UPDATE")
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select promotion for discount: %w", err)
	}
	if input.SubtotalMinor < promotion.MinimumAmount || input.Nights < promotion.MinimumNights {
		return nil, nil
	}
	amount := promotion.Value
	if promotion.Type != domain.TypeFixedAmount {
		amount = int64(math.Round(float64(input.SubtotalMinor) * float64(promotion.Value) / 100))
	}
	if amount > input.SubtotalMinor {
		amount = input.SubtotalMinor
	}
	if amount <= 0 {
		return nil, nil
	}
	if _, err = tx.Exec(ctx, `UPDATE business_promotions SET usage_count=usage_count+1 WHERE id=$1`, promotion.ID); err != nil {
		return nil, fmt.Errorf("record promotion use: %w", err)
	}
	if err = refreshBusinessPromotion(ctx, tx, input.BusinessID); err != nil {
		return nil, err
	}
	return &domain.AppliedDiscount{PromotionID: promotion.ID, AmountMinor: amount}, nil
}
