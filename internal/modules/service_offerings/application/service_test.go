package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/domain"
	"testing"
)

func TestCreateRejectsInvalidDuration(t *testing.T) {
	t.Parallel()
	_, e := NewService(&fakeRepository{}, &fakeQueries{}).Create(context.Background(), "p", "provider", uuid.New(), domain.CreateInput{Name: "Cut", DurationMinutes: 4})
	if !errors.Is(e, ErrValidation) {
		t.Fatalf("expected validation, got %v", e)
	}
}
func TestCreateDefaultsActive(t *testing.T) {
	t.Parallel()
	r := &fakeRepository{}
	_, e := NewService(r, &fakeQueries{}).Create(context.Background(), "p", "provider", uuid.New(), domain.CreateInput{Name: "Cut", DurationMinutes: 30, PriceMinor: 1000})
	if e != nil {
		t.Fatal(e)
	}
	if r.input.IsActive == nil || !*r.input.IsActive {
		t.Fatal("expected active default")
	}
}

type fakeRepository struct {
	input                      domain.CreateInput
	created, updated, archived bool
}

func (r *fakeRepository) CreateOwned(_ context.Context, _ string, b uuid.UUID, i domain.CreateInput) (domain.Offering, error) {
	r.input = i
	r.created = true
	return domain.Offering{BusinessID: b}, nil
}
func (r *fakeRepository) UpdateOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.UpdateInput) (domain.Offering, error) {
	r.updated = true
	return domain.Offering{}, nil
}
func (r *fakeRepository) ArchiveOwned(context.Context, string, uuid.UUID, uuid.UUID) error {
	r.archived = true
	return nil
}

type fakeQueries struct{ listed bool }

func (q *fakeQueries) ListOwned(context.Context, string, uuid.UUID) ([]domain.Offering, error) {
	q.listed = true
	return nil, nil
}

func TestOperationsUseTheirDedicatedPorts(t *testing.T) {
	t.Parallel()
	repository, queries := &fakeRepository{}, &fakeQueries{}
	service := NewService(repository, queries)
	businessID, offeringID := uuid.New(), uuid.New()
	if _, err := service.List(context.Background(), "provider", "provider", businessID); err != nil || !queries.listed {
		t.Fatalf("list should use query port, err=%v listed=%v", err, queries.listed)
	}
	if _, err := service.Create(context.Background(), "provider", "provider", businessID, domain.CreateInput{Name: "Cut", DurationMinutes: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(context.Background(), "provider", "provider", businessID, offeringID, domain.UpdateInput{IsActive: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	if err := service.Archive(context.Background(), "provider", "provider", businessID, offeringID); err != nil {
		t.Fatal(err)
	}
	if !repository.created || !repository.updated || !repository.archived {
		t.Fatalf("commands should use command repository: %+v", repository)
	}
}

func boolPtr(value bool) *bool { return &value }
