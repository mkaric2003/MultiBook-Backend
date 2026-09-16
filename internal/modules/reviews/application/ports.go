package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
)

type Repository interface {
	Create(ctx context.Context, customerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Review, error)
}

type Queries interface {
	HasReview(ctx context.Context, customerID string, businessID uuid.UUID) (bool, error)
	List(ctx context.Context, businessID uuid.UUID, limit, offset int) ([]domain.Review, error)
}
