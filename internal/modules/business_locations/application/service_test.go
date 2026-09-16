package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_locations/domain"
)

func TestUpsertUsesCommandRepository(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	_, err := NewService(repository).Upsert(context.Background(), "provider", "provider", uuid.New(), domain.UpsertInput{City: "Sarajevo", Address: "Main 1", Latitude: 43.8563, Longitude: 18.4131})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.upserted {
		t.Fatal("upsert should use command repository")
	}
}

func TestUpsertNormalizesBusinessLocationTextWithoutMutatingCaller(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	city, address, countryCode := "  Sarajevo  ", "  Main 1  ", " BA "
	input := domain.UpsertInput{City: city, Address: address, CountryCode: &countryCode, Latitude: 43.8563, Longitude: 18.4131}
	location, err := NewService(repository).Upsert(context.Background(), "provider", "provider", uuid.New(), input)
	if err != nil {
		t.Fatal(err)
	}
	if location.City != "Sarajevo" || location.Address != "Main 1" || location.CountryCode == nil || *location.CountryCode != "BA" {
		t.Fatalf("normalized location = %#v", location)
	}
	if input.City != city || input.Address != address || *input.CountryCode != countryCode {
		t.Fatalf("Upsert() mutated caller input: %#v", input)
	}
}

type fakeRepository struct{ upserted bool }

func (r *fakeRepository) UpsertOwned(_ context.Context, _ string, _ uuid.UUID, input domain.UpsertInput) (domain.Location, error) {
	r.upserted = true
	return domain.Location{City: input.City, Address: input.Address, CountryCode: input.CountryCode}, nil
}
