package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
)

type CommandRepository interface {
	Create(context.Context, string, uuid.UUID, domain.CreateInput) (domain.Promotion, error)
	SetActive(context.Context, string, uuid.UUID, uuid.UUID, bool) (domain.Promotion, error)
	Delete(context.Context, string, uuid.UUID, uuid.UUID) error
}

type Queries interface {
	ListOwned(context.Context, string, uuid.UUID) ([]domain.Promotion, error)
	Active(context.Context, uuid.UUID, *string) (*domain.Promotion, error)
}

type DiscountApplier interface {
	Apply(context.Context, pgx.Tx, domain.DiscountInput) (*domain.AppliedDiscount, error)
}
