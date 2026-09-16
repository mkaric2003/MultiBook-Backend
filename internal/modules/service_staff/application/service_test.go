package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/domain"
)

func TestCreateRejectsCustomer(t *testing.T) {
	t.Parallel()
	_, err := NewService(&fakeRepository{}, &fakeQueries{}).Create(context.Background(), "customer", "customer", uuid.New(), validInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestCreateDefaultsCommissionAndActivity(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	_, err := NewService(repository, &fakeQueries{}).Create(context.Background(), "provider", "provider", uuid.New(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	if *repository.input.CommissionRate != 100 || !*repository.input.IsActive {
		t.Fatal("expected defaults")
	}
}

func TestUpdateRejectsInvalidCommission(t *testing.T) {
	t.Parallel()
	value := 101.0
	_, err := NewService(&fakeRepository{}, &fakeQueries{}).Update(context.Background(), "provider", "provider", uuid.New(), uuid.New(), domain.UpdateInput{CommissionRate: &value})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func validInput() domain.CreateInput { return domain.CreateInput{Name: "Amina"} }

type fakeRepository struct {
	input                      domain.CreateInput
	created, updated, archived bool
}

func (r *fakeRepository) CreateOwned(_ context.Context, _ string, businessID uuid.UUID, input domain.CreateInput) (domain.Staff, error) {
	r.input = input
	r.created = true
	return domain.Staff{BusinessID: businessID}, nil
}
func (r *fakeRepository) UpdateOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.UpdateInput) (domain.Staff, error) {
	r.updated = true
	return domain.Staff{}, nil
}
func (r *fakeRepository) ArchiveOwned(context.Context, string, uuid.UUID, uuid.UUID) error {
	r.archived = true
	return nil
}

type fakeQueries struct{ listed bool }

func (q *fakeQueries) ListOwned(context.Context, string, uuid.UUID) ([]domain.Staff, error) {
	q.listed = true
	return nil, nil
}

func TestOperationsUseTheirDedicatedPorts(t *testing.T) {
	t.Parallel()
	repository, queries := &fakeRepository{}, &fakeQueries{}
	service := NewService(repository, queries)
	businessID, staffID := uuid.New(), uuid.New()
	if _, err := service.List(context.Background(), "provider", "provider", businessID); err != nil || !queries.listed {
		t.Fatalf("list should use query port, err=%v listed=%v", err, queries.listed)
	}
	if _, err := service.Create(context.Background(), "provider", "provider", businessID, validInput()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(context.Background(), "provider", "provider", businessID, staffID, domain.UpdateInput{IsActive: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	if err := service.Archive(context.Background(), "provider", "provider", businessID, staffID); err != nil {
		t.Fatal(err)
	}
	if !repository.created || !repository.updated || !repository.archived {
		t.Fatalf("commands should use command repository: %+v", repository)
	}
}

func boolPtr(value bool) *bool { return &value }
