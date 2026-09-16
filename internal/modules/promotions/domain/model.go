// Package domain contains promotion models without transport or persistence dependencies.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypePercentage  Type = "percentage"
	TypeFixedAmount Type = "fixedAmount"
	TypeCouponCode  Type = "couponCode"
)

func (promotionType Type) Valid() bool {
	return promotionType == TypePercentage || promotionType == TypeFixedAmount || promotionType == TypeCouponCode
}

type Promotion struct {
	ID            uuid.UUID
	BusinessID    uuid.UUID
	OwnerID       string
	Name          string
	Type          Type
	Value         int64
	StartsAt      time.Time
	EndsAt        time.Time
	IsActive      bool
	Code          *string
	MinimumAmount int64
	MinimumNights int
	UsageLimit    *int
	UsageCount    int
	CreatedAt     time.Time
}

type CreateInput struct {
	Name          string
	Type          Type
	Value         int64
	StartsAt      time.Time
	EndsAt        time.Time
	Code          *string
	MinimumAmount int64
	MinimumNights int
	UsageLimit    *int
}

type DiscountInput struct {
	BusinessID    uuid.UUID
	PromoCode     *string
	SubtotalMinor int64
	Nights        int
}

type AppliedDiscount struct {
	PromotionID uuid.UUID
	AmountMinor int64
}
