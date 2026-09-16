package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/domain"
	notificationsdomain "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

func TestCreateRejectsNonCustomer(t *testing.T) {
	repository := &fakeRepository{}
	_, err := NewService(repository, &queryStub{}).Create(context.Background(), "provider", "provider", uuid.New(), validInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
func TestCreateRequiresThirtyMinuteBoundary(t *testing.T) {
	input := validInput()
	input.StartMinutes = 15
	repository := &fakeRepository{}
	_, err := NewService(repository, &queryStub{}).Create(context.Background(), "customer", "customer", uuid.New(), input)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}
func TestCreateNormalizesCustomerInput(t *testing.T) {
	repository := &fakeRepository{}
	_, err := NewService(repository, &queryStub{}).Create(context.Background(), "customer", "customer", uuid.New(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	if repository.input.CustomerName != "Amina" {
		t.Fatalf("name = %q", repository.input.CustomerName)
	}
}
func TestAvailabilityRejectsDuplicateOfferings(t *testing.T) {
	id := uuid.New()
	repository := &fakeRepository{}
	_, err := NewService(repository, &queryStub{}).Availability(context.Background(), "user", uuid.New(), uuid.New(), domain.AvailabilityInput{AppointmentDate: "2026-09-01", OfferingIDs: []uuid.UUID{id, id}})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func TestReadUseCasesUseAppointmentQueries(t *testing.T) {
	queries := &queryStub{}
	service := NewService(&fakeRepository{}, queries)
	if _, err := service.List(context.Background(), "customer", "customer", domain.ListInput{}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.Availability(context.Background(), "customer", uuid.New(), uuid.New(), domain.AvailabilityInput{AppointmentDate: "2026-09-01", OfferingIDs: []uuid.UUID{uuid.New()}}); err != nil {
		t.Fatalf("Availability() error = %v", err)
	}
	if !queries.listCalled || !queries.availabilityCalled {
		t.Fatalf("read use cases did not use query port: %#v", queries)
	}
}

func TestListNormalizesPagination(t *testing.T) {
	queries := &queryStub{}
	service := NewService(&fakeRepository{}, queries)
	if _, err := service.List(context.Background(), "customer", "customer", domain.ListInput{Limit: 100}); err != nil {
		t.Fatal(err)
	}
	if queries.listInput.Limit != 50 {
		t.Fatalf("limit = %d, want 50", queries.listInput.Limit)
	}
}

func TestStatusPublishesOnlyProviderNotificationAfterCustomerCancellation(t *testing.T) {
	appointmentID := uuid.New()
	repository := &fakeRepository{statusAppointment: domain.Appointment{
		ID:              appointmentID,
		BusinessID:      uuid.New(),
		BusinessOwnerID: "provider",
		CustomerID:      "customer",
		BusinessName:    "Studio",
		Status:          domain.StatusCancelled,
	}}
	publisher := &publisherStub{}

	_, err := NewService(repository, &queryStub{}, publisher).Status(context.Background(), "customer", "customer", appointmentID, domain.StatusCancelled)
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	providerEvent := publisher.events[0]
	if providerEvent.RecipientID != "provider" || providerEvent.Kind != "appointment_cancelled_by_customer" || providerEvent.Data["customerId"] != "customer" {
		t.Fatalf("unexpected provider notification event: %#v", providerEvent)
	}
}
func validInput() domain.CreateInput {
	return domain.CreateInput{StaffID: uuid.New(), AppointmentDate: "2026-09-01", StartMinutes: 600, OfferingIDs: []uuid.UUID{uuid.New()}, CustomerName: " Amina ", CustomerEmail: " a@example.test ", CustomerPhone: " +38761123456 ", PaymentMethod: " cash "}
}

type fakeRepository struct {
	input             domain.CreateInput
	statusAppointment domain.Appointment
}

func (r *fakeRepository) CreateForCustomer(_ context.Context, _ string, _ uuid.UUID, input domain.CreateInput) (domain.Appointment, error) {
	r.input = input
	return domain.Appointment{}, nil
}

type queryStub struct {
	availabilityCalled bool
	listCalled         bool
	listInput          domain.ListInput
}

func (r *queryStub) AvailableSlots(context.Context, uuid.UUID, uuid.UUID, domain.AvailabilityInput) (domain.AvailableSlots, error) {
	r.availabilityCalled = true
	return domain.AvailableSlots{}, nil
}
func (r *queryStub) ListForActor(_ context.Context, _ string, _ string, input domain.ListInput) (domain.Page, error) {
	r.listCalled = true
	r.listInput = input
	return domain.Page{}, nil
}
func (r *fakeRepository) SetStatus(context.Context, string, string, uuid.UUID, domain.Status) (domain.Appointment, error) {
	return r.statusAppointment, nil
}
func (*fakeRepository) Reschedule(context.Context, string, string, uuid.UUID, domain.RescheduleInput) (domain.Appointment, error) {
	return domain.Appointment{}, nil
}

type publisherStub struct{ events []notificationsdomain.Event }

func (p *publisherStub) Publish(_ context.Context, event notificationsdomain.Event) {
	p.events = append(p.events, event)
}
