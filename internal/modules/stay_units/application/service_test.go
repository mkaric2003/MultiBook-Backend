package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/domain"
)

func TestCreateNormalizesNameAndDefaultsActivity(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	service := NewService(repository, &fakeQueries{})

	_, err := service.Create(context.Background(), "provider-1", "provider", uuid.New(), domain.CreateInput{
		Name:               "  Deluxe room  ",
		MaxGuests:          2,
		SizeSquareMeters:   32,
		PricePerNightMinor: 12000,
		Quantity:           3,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if repository.input.Name != "Deluxe room" {
		t.Fatalf("name = %q", repository.input.Name)
	}
	if repository.input.IsActive == nil || !*repository.input.IsActive {
		t.Fatal("expected is_active to default to true")
	}
}

func TestCreateRejectsNonProvider(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepository{}, &fakeQueries{})

	_, err := service.Create(context.Background(), "customer-1", "customer", uuid.New(), validCreateInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestUpdateRejectsInvalidQuantity(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepository{}, &fakeQueries{})
	quantity := int16(0)

	_, err := service.Update(context.Background(), "provider-1", "provider", uuid.New(), uuid.New(), domain.UpdateInput{Quantity: &quantity})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func validCreateInput() domain.CreateInput {
	return domain.CreateInput{
		Name:               "Deluxe room",
		MaxGuests:          2,
		SizeSquareMeters:   32,
		PricePerNightMinor: 12000,
		Quantity:           3,
	}
}

type fakeRepository struct {
	input                      domain.CreateInput
	created, updated, archived bool
}

func (r *fakeRepository) CreateOwned(_ context.Context, _ string, businessID uuid.UUID, input domain.CreateInput) (domain.UnitType, error) {
	r.input = input
	r.created = true
	return domain.UnitType{BusinessID: businessID}, nil
}

func (r *fakeRepository) UpdateOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.UpdateInput) (domain.UnitType, error) {
	r.updated = true
	return domain.UnitType{}, nil
}

func (r *fakeRepository) ArchiveOwned(context.Context, string, uuid.UUID, uuid.UUID) error {
	r.archived = true
	return nil
}

type fakeQueries struct{ listed bool }

func (q *fakeQueries) ListOwned(context.Context, string, uuid.UUID) ([]domain.UnitType, error) {
	q.listed = true
	return nil, nil
}

func TestOperationsUseTheirDedicatedPorts(t *testing.T) {
	t.Parallel()
	repository, queries := &fakeRepository{}, &fakeQueries{}
	service := NewService(repository, queries)
	businessID, unitTypeID := uuid.New(), uuid.New()
	if _, err := service.List(context.Background(), "provider", "provider", businessID); err != nil || !queries.listed {
		t.Fatalf("list should use query port, err=%v listed=%v", err, queries.listed)
	}
	if _, err := service.Create(context.Background(), "provider", "provider", businessID, validCreateInput()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(context.Background(), "provider", "provider", businessID, unitTypeID, domain.UpdateInput{IsActive: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	if err := service.Archive(context.Background(), "provider", "provider", businessID, unitTypeID); err != nil {
		t.Fatal(err)
	}
	if !repository.created || !repository.updated || !repository.archived {
		t.Fatalf("commands should use command repository: %+v", repository)
	}
}

func boolPtr(value bool) *bool { return &value }
