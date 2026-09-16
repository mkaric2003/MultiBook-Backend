package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	discoveryapp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

type savedStub struct {
	customerID string
	businessID uuid.UUID
	ids        []uuid.UUID
	calls      int
	err        error
}

func (s *savedStub) capture(customerID string, businessID uuid.UUID) error {
	s.customerID, s.businessID = customerID, businessID
	s.calls++
	return s.err
}
func (s *savedStub) Save(_ context.Context, customerID string, businessID uuid.UUID) error {
	return s.capture(customerID, businessID)
}
func (s *savedStub) Remove(_ context.Context, customerID string, businessID uuid.UUID) error {
	return s.capture(customerID, businessID)
}
func (s *savedStub) IsSaved(_ context.Context, customerID string, businessID uuid.UUID) (bool, error) {
	return true, s.capture(customerID, businessID)
}
func (s *savedStub) ListBusinessIDs(_ context.Context, customerID string) ([]uuid.UUID, error) {
	s.customerID, s.calls = customerID, s.calls+1
	return s.ids, s.err
}
func (s *savedStub) ListActiveByIDs(_ context.Context, ids []uuid.UUID) ([]discoveryapp.CustomerBusinessSummary, error) {
	s.ids, s.calls = ids, s.calls+1
	return nil, s.err
}

func TestSavedAuthorizationAndScoping(t *testing.T) {
	id := uuid.New()
	for _, operation := range []string{"save", "remove", "check", "list"} {
		t.Run(operation, func(t *testing.T) {
			stub := &savedStub{}
			service := NewService(stub, stub, stub)
			call := func(actor Actor, id uuid.UUID) error {
				switch operation {
				case "save":
					return service.Save(context.Background(), actor, id)
				case "remove":
					return service.Remove(context.Background(), actor, id)
				case "check":
					_, err := service.IsSaved(context.Background(), actor, id)
					return err
				default:
					_, err := service.List(context.Background(), actor)
					return err
				}
			}
			for _, actor := range []Actor{{}, {ID: "provider", Role: "provider"}, {ID: "admin", Role: "admin"}, {Role: "customer"}} {
				if err := call(actor, id); !errors.Is(err, ErrForbidden) {
					t.Fatalf("actor %#v: %v", actor, err)
				}
			}
			if stub.calls != 0 {
				t.Fatal("unauthorized persistence access")
			}
			customer := Actor{ID: "customer-a", Role: "customer"}
			if operation != "list" {
				if err := call(customer, uuid.Nil); !errors.Is(err, ErrValidation) {
					t.Fatalf("nil id: %v", err)
				}
				if stub.calls != 0 {
					t.Fatal("invalid id reached persistence")
				}
			}
			if err := call(customer, id); err != nil {
				t.Fatal(err)
			}
			if stub.customerID != customer.ID || (operation != "list" && stub.businessID != id) {
				t.Fatal("request not scoped to authenticated customer")
			}
		})
	}
}

func TestListHydratesSavedIDsThroughDiscoveryReader(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	relations := &savedStub{ids: []uuid.UUID{first, second}}
	summaries := &savedStub{}
	_, err := NewService(relations, relations, summaries).List(context.Background(), Actor{ID: "customer", Role: "customer"})
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries.ids) != 2 || summaries.ids[0] != first || summaries.ids[1] != second {
		t.Fatalf("saved ordering was not passed to discovery: %v", summaries.ids)
	}
}
