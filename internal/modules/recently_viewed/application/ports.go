package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// Repository persists recently viewed business relations.
type Repository interface {
	Record(ctx context.Context, customerID string, businessID uuid.UUID) error
}

// Queries reads relation identities in display order. Business snapshots stay
// owned by customer_discovery and are hydrated through its published reader.
type Queries interface {
	ListBusinessIDs(ctx context.Context, customerID string, businessType domain.BusinessType, limit int) ([]uuid.UUID, error)
}
