package application

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists Saved relation commands.
type Repository interface {
	Save(context.Context, string, uuid.UUID) error
	Remove(context.Context, string, uuid.UUID) error
}

// Queries reads only the customer-owned Saved relation. Business projection
// hydration is delegated to customer_discovery.
type Queries interface {
	IsSaved(context.Context, string, uuid.UUID) (bool, error)
	ListBusinessIDs(context.Context, string) ([]uuid.UUID, error)
}
