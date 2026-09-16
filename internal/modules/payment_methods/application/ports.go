package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/domain"
)

type CommandRepository interface {
	Create(context.Context, string, domain.CreateInput) (domain.PaymentMethod, error)
	SetDefault(context.Context, string, uuid.UUID) (domain.PaymentMethod, error)
	Delete(context.Context, string, uuid.UUID) error
}

type Queries interface {
	List(context.Context, string) ([]domain.PaymentMethod, error)
}
