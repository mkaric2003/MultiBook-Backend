package http

import (
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
)

type promotionResponse struct {
	ID            uuid.UUID   `json:"id"`
	BusinessID    uuid.UUID   `json:"businessId"`
	OwnerID       string      `json:"ownerId"`
	Name          string      `json:"name"`
	Type          domain.Type `json:"type"`
	Value         int64       `json:"value"`
	StartsAt      time.Time   `json:"startsAt"`
	EndsAt        time.Time   `json:"endsAt"`
	IsActive      bool        `json:"isActive"`
	Code          *string     `json:"code"`
	MinimumAmount int64       `json:"minimumAmount"`
	MinimumNights int         `json:"minimumNights"`
	UsageLimit    *int        `json:"usageLimit"`
	UsageCount    int         `json:"usageCount"`
	CreatedAt     time.Time   `json:"createdAt"`
}

func response(promotion domain.Promotion) promotionResponse {
	return promotionResponse{
		ID: promotion.ID, BusinessID: promotion.BusinessID, OwnerID: promotion.OwnerID,
		Name: promotion.Name, Type: promotion.Type, Value: promotion.Value,
		StartsAt: promotion.StartsAt, EndsAt: promotion.EndsAt, IsActive: promotion.IsActive,
		Code: promotion.Code, MinimumAmount: promotion.MinimumAmount, MinimumNights: promotion.MinimumNights,
		UsageLimit: promotion.UsageLimit, UsageCount: promotion.UsageCount, CreatedAt: promotion.CreatedAt,
	}
}
