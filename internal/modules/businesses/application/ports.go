package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// BusinessRepository persists business commands and ownership changes.
type BusinessRepository interface {
	Create(ctx context.Context, actor Actor, input domain.CreateInput) (PersistedBusiness, error)
	ReplaceOwned(ctx context.Context, actor Actor, id uuid.UUID, input domain.CreateInput) (PersistedBusiness, error)
	UpdateOwned(ctx context.Context, actor Actor, id uuid.UUID, input domain.UpdateInput) (domain.Business, error)
	ArchiveOwned(ctx context.Context, actor Actor, id uuid.UUID) error
	SetSelectedBusiness(ctx context.Context, actor Actor, id uuid.UUID) error
}

// PersistedBusiness contains DB-generated aggregate identities. The
// application layer applies these to a copy of the command input when a
// transport response needs the complete created aggregate.
type PersistedBusiness struct {
	Business     domain.Business
	AggregateIDs AggregateIDMapping
}

type AggregateIDMapping struct {
	StayUnitTypeIDs             []uuid.UUID
	ServiceOfferingIDs          []uuid.UUID
	ServiceStaffIDs             []uuid.UUID
	ServiceStaffAvailabilityIDs [][]uuid.UUID
}

type CreatedBusiness struct {
	Business domain.Business
	Input    domain.CreateInput
}

// BusinessQueries provides optimized typed read models for business screens.
type BusinessQueries interface {
	ListOwned(ctx context.Context, ownerID string) ([]OwnedSummary, error)
	GetOwnedDetail(ctx context.Context, ownerID string, id uuid.UUID) (OwnedDetail, error)
}
