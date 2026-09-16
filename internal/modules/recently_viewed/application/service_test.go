package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	discoveryapp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

type relationStub struct {
	customerID   string
	businessID   uuid.UUID
	businessType domain.BusinessType
	limit        int
	ids          []uuid.UUID
	recordCalls  int
	queryCalls   int
	err          error
}

func (s *relationStub) Record(_ context.Context, customerID string, businessID uuid.UUID) error {
	s.customerID, s.businessID = customerID, businessID
	s.recordCalls++
	return s.err
}

func (s *relationStub) ListBusinessIDs(_ context.Context, customerID string, businessType domain.BusinessType, limit int) ([]uuid.UUID, error) {
	s.customerID, s.businessType, s.limit = customerID, businessType, limit
	s.queryCalls++
	return s.ids, s.err
}

type summaryStub struct {
	ids   []uuid.UUID
	calls int
	err   error
}

func (s *summaryStub) ListActiveByIDs(_ context.Context, ids []uuid.UUID) ([]discoveryapp.CustomerBusinessSummary, error) {
	s.ids, s.calls = ids, s.calls+1
	return nil, s.err
}

func TestRecordAuthorizationValidationAndScoping(t *testing.T) {
	businessID := uuid.New()
	for _, actor := range []Actor{{}, {ID: "provider", Role: "provider"}, {ID: "admin", Role: "admin"}, {Role: "customer"}} {
		relations := &relationStub{}
		err := NewService(relations, relations, &summaryStub{}).Record(context.Background(), actor, businessID)
		if !errors.Is(err, ErrForbidden) || relations.recordCalls != 0 {
			t.Fatalf("actor %#v: error=%v calls=%d", actor, err, relations.recordCalls)
		}
	}
	relations := &relationStub{}
	service := NewService(relations, relations, &summaryStub{})
	customer := Actor{ID: "customer-a", Role: "customer"}
	if err := service.Record(context.Background(), customer, uuid.Nil); !errors.Is(err, ErrValidation) {
		t.Fatalf("nil business ID: %v", err)
	}
	if relations.recordCalls != 0 {
		t.Fatal("invalid business ID reached persistence")
	}
	if err := service.Record(context.Background(), customer, businessID); err != nil {
		t.Fatal(err)
	}
	if relations.customerID != customer.ID || relations.businessID != businessID {
		t.Fatal("record was not scoped to the authenticated customer")
	}
}

func TestListAuthorizationTypeAndLimit(t *testing.T) {
	customer := Actor{ID: "customer-a", Role: "customer"}
	for _, actor := range []Actor{{}, {ID: "provider", Role: "provider"}, {Role: "customer"}} {
		relations, summaries := &relationStub{}, &summaryStub{}
		_, err := NewService(relations, relations, summaries).List(context.Background(), actor, domain.BusinessTypeStay, 10)
		if !errors.Is(err, ErrForbidden) || relations.queryCalls != 0 || summaries.calls != 0 {
			t.Fatalf("actor %#v: error=%v queryCalls=%d summaryCalls=%d", actor, err, relations.queryCalls, summaries.calls)
		}
	}
	relations, summaries := &relationStub{}, &summaryStub{}
	service := NewService(relations, relations, summaries)
	if _, err := service.List(context.Background(), customer, "invalid", 10); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid type: %v", err)
	}
	for _, test := range []struct {
		requested int
		want      int
	}{{0, 10}, {-1, 10}, {5, 5}, {31, 30}} {
		if _, err := service.List(context.Background(), customer, domain.BusinessTypeStay, test.requested); err != nil {
			t.Fatal(err)
		}
		if relations.limit != test.want {
			t.Fatalf("limit %d normalized to %d, want %d", test.requested, relations.limit, test.want)
		}
	}
}

func TestListHydratesRelationIDsThroughDiscoveryReader(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	relations := &relationStub{ids: []uuid.UUID{first, second}}
	summaries := &summaryStub{}
	_, err := NewService(relations, relations, summaries).List(
		context.Background(),
		Actor{ID: "customer", Role: "customer"},
		domain.BusinessTypeService,
		7,
	)
	if err != nil {
		t.Fatal(err)
	}
	if relations.customerID != "customer" || relations.businessType != domain.BusinessTypeService || relations.limit != 7 {
		t.Fatalf("incorrect relation query: %#v", relations)
	}
	if len(summaries.ids) != 2 || summaries.ids[0] != first || summaries.ids[1] != second {
		t.Fatalf("relation ordering was not passed to discovery: %v", summaries.ids)
	}
}
