package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/domain"
)

func TestUpsertRejectsUnsupportedAmenity(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepository{}, &fakeQueries{})
	price := int64(12000)

	_, err := service.Upsert(context.Background(), "provider-1", "provider", uuid.New(), domain.UpsertInput{
		InventoryType:  domain.InventoryTypeSingleUnit,
		BasePriceMinor: &price,
		Amenities:      []string{"teleport"},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpsertNormalizesAndDeduplicatesAmenities(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	service := NewService(repository, &fakeQueries{})
	price := int64(12000)
	businessID := uuid.New()

	_, err := service.Upsert(context.Background(), "provider-1", "provider", businessID, domain.UpsertInput{
		InventoryType:  domain.InventoryTypeSingleUnit,
		BasePriceMinor: &price,
		Amenities:      []string{"parking", "wifi", "parking"},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if got, want := repository.input.Amenities, []string{"parking", "wifi"}; !equalStrings(got, want) {
		t.Fatalf("amenities = %v, want %v", got, want)
	}
}

func TestUpsertSingleUnitRequiresBasePrice(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepository{}, &fakeQueries{})

	_, err := service.Upsert(context.Background(), "provider-1", "provider", uuid.New(), domain.UpsertInput{
		InventoryType: domain.InventoryTypeSingleUnit,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpsertRequiresProviderRole(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepository{}, &fakeQueries{})
	price := int64(12000)

	_, err := service.Upsert(context.Background(), "customer-1", "customer", uuid.New(), domain.UpsertInput{
		InventoryType:  domain.InventoryTypeSingleUnit,
		BasePriceMinor: &price,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

type fakeRepository struct {
	input    domain.UpsertInput
	upserted bool
}

func (r *fakeRepository) UpsertOwned(_ context.Context, _ string, businessID uuid.UUID, input domain.UpsertInput) (domain.Details, error) {
	r.input = input
	r.upserted = true
	return domain.Details{BusinessID: businessID, Amenities: input.Amenities}, nil
}

type fakeQueries struct{ fetched bool }

func (q *fakeQueries) GetOwned(context.Context, string, uuid.UUID) (domain.Details, error) {
	q.fetched = true
	return domain.Details{}, nil
}

func TestOperationsUseTheirDedicatedPorts(t *testing.T) {
	t.Parallel()
	repository, queries := &fakeRepository{}, &fakeQueries{}
	service := NewService(repository, queries)
	businessID := uuid.New()
	if _, err := service.GetOwned(context.Background(), "provider", "provider", businessID); err != nil || !queries.fetched {
		t.Fatalf("get should use query port, err=%v fetched=%v", err, queries.fetched)
	}
	price := int64(12000)
	if _, err := service.Upsert(context.Background(), "provider", "provider", businessID, domain.UpsertInput{InventoryType: domain.InventoryTypeSingleUnit, BasePriceMinor: &price}); err != nil {
		t.Fatal(err)
	}
	if !repository.upserted {
		t.Fatal("upsert should use command repository")
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
