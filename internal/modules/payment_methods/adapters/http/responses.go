package http

import (
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/domain"
)

type paymentMethodResponse struct {
	ID          uuid.UUID        `json:"id"`
	UserID      string           `json:"userId"`
	Brand       domain.CardBrand `json:"brand"`
	Last4       string           `json:"last4"`
	ExpiryMonth int              `json:"expiryMonth"`
	ExpiryYear  int              `json:"expiryYear"`
	HolderName  string           `json:"holderName"`
	IsDefault   bool             `json:"isDefault"`
	CreatedAt   time.Time        `json:"createdAt"`
}

func response(method domain.PaymentMethod) paymentMethodResponse {
	return paymentMethodResponse{
		ID: method.ID, UserID: method.CustomerID, Brand: method.Brand, Last4: method.Last4,
		ExpiryMonth: method.ExpiryMonth, ExpiryYear: method.ExpiryYear,
		HolderName: method.HolderName, IsDefault: method.IsDefault, CreatedAt: method.CreatedAt,
	}
}
