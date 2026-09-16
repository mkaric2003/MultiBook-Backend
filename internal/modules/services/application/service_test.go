package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/domain"
	"testing"
)

func TestUpsertRejectsInvalidTimeZone(t *testing.T) {
	t.Parallel()
	_, e := NewService(&fakeRepository{}, &fakeQueries{}).Upsert(context.Background(), "p", "provider", uuid.New(), domain.UpsertInput{TimeZone: "not/a-zone"})
	if !errors.Is(e, ErrValidation) {
		t.Fatalf("expected validation, got %v", e)
	}
}
func TestUpsertRejectsCustomer(t *testing.T) {
	t.Parallel()
	_, e := NewService(&fakeRepository{}, &fakeQueries{}).Upsert(context.Background(), "c", "customer", uuid.New(), domain.UpsertInput{TimeZone: "Europe/Sarajevo"})
	if !errors.Is(e, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", e)
	}
}

type fakeRepository struct{ upserted bool }

func (r *fakeRepository) UpsertOwned(context.Context, string, uuid.UUID, domain.UpsertInput) (domain.Details, error) {
	r.upserted = true
	return domain.Details{}, nil
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
	if _, err := service.Get(context.Background(), "provider", "provider", businessID); err != nil || !queries.fetched {
		t.Fatalf("get should use query port, err=%v fetched=%v", err, queries.fetched)
	}
	if _, err := service.Upsert(context.Background(), "provider", "provider", businessID, domain.UpsertInput{TimeZone: "Europe/Sarajevo"}); err != nil {
		t.Fatal(err)
	}
	if !repository.upserted {
		t.Fatal("upsert should use command repository")
	}
}
