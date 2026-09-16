package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/domain"
	notificationsapp "github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	notificationsdomain "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

var (
	ErrNotFound   = errors.New("appointment resource not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
	ErrConflict   = errors.New("appointment time is unavailable")
)

type AppointmentRepository interface {
	CreateForCustomer(context.Context, string, uuid.UUID, domain.CreateInput) (domain.Appointment, error)
	SetStatus(context.Context, string, string, uuid.UUID, domain.Status) (domain.Appointment, error)
	Reschedule(context.Context, string, string, uuid.UUID, domain.RescheduleInput) (domain.Appointment, error)
}

type AppointmentQueries interface {
	AvailableSlots(context.Context, uuid.UUID, uuid.UUID, domain.AvailabilityInput) (domain.AvailableSlots, error)
	ListForActor(context.Context, string, string, domain.ListInput) (domain.Page, error)
}

func (s *Service) List(ctx context.Context, actorID, role string, input domain.ListInput) (domain.Page, error) {
	if strings.TrimSpace(actorID) == "" {
		return domain.Page{}, fmt.Errorf("%w: authentication is required", ErrForbidden)
	}
	if role != "customer" && role != "provider" {
		return domain.Page{}, fmt.Errorf("%w: customer or provider role is required", ErrForbidden)
	}
	if input.Offset < 0 {
		return domain.Page{}, invalid("cursor is invalid")
	}
	if input.Limit < 1 {
		input.Limit = 20
	}
	if input.Limit > 50 {
		input.Limit = 50
	}
	return s.queries.ListForActor(ctx, actorID, role, input)
}
func (s *Service) Status(ctx context.Context, actorID, role string, appointmentID uuid.UUID, status domain.Status) (domain.Appointment, error) {
	if strings.TrimSpace(actorID) == "" {
		return domain.Appointment{}, fmt.Errorf("%w: authentication is required", ErrForbidden)
	}
	if status != domain.StatusCancelled && status != domain.StatusDeclined && status != domain.StatusCompleted && status != domain.StatusNoShow {
		return domain.Appointment{}, invalid("unsupported appointment status")
	}
	appointment, err := s.repository.SetStatus(ctx, actorID, role, appointmentID, status)
	if err == nil {
		if role == "customer" && status == domain.StatusCancelled {
			s.publishProviderCancellation(ctx, appointment)
		} else {
			s.publishCustomerStatus(ctx, appointment)
		}
	}
	return appointment, err
}
func (s *Service) Reschedule(ctx context.Context, actorID, role string, appointmentID uuid.UUID, input domain.RescheduleInput) (domain.Appointment, error) {
	if strings.TrimSpace(actorID) == "" {
		return domain.Appointment{}, fmt.Errorf("%w: authentication is required", ErrForbidden)
	}
	if _, err := time.Parse(time.DateOnly, input.AppointmentDate); err != nil {
		return domain.Appointment{}, invalid("appointment_date must be an ISO date")
	}
	if input.StartMinutes < 0 || input.StartMinutes >= 1440 || input.StartMinutes%30 != 0 {
		return domain.Appointment{}, invalid("start_minutes must be a 30-minute boundary between 0 and 1439")
	}
	return s.repository.Reschedule(ctx, actorID, role, appointmentID, input)
}

func (s *Service) Availability(ctx context.Context, actorID string, businessID, staffID uuid.UUID, input domain.AvailabilityInput) (domain.AvailableSlots, error) {
	if strings.TrimSpace(actorID) == "" {
		return domain.AvailableSlots{}, fmt.Errorf("%w: authentication is required", ErrForbidden)
	}
	if _, err := time.Parse(time.DateOnly, input.AppointmentDate); err != nil {
		return domain.AvailableSlots{}, invalid("appointment_date must be an ISO date")
	}
	if staffID == uuid.Nil || len(input.OfferingIDs) == 0 {
		return domain.AvailableSlots{}, invalid("staff_id and at least one offering_id are required")
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range input.OfferingIDs {
		if id == uuid.Nil || seen[id] {
			return domain.AvailableSlots{}, invalid("offering_ids must be unique UUIDs")
		}
		seen[id] = true
	}
	return s.queries.AvailableSlots(ctx, businessID, staffID, input)
}

type Service struct {
	repository    AppointmentRepository
	queries       AppointmentQueries
	notifications notificationsapp.Publisher
}

func NewService(repository AppointmentRepository, queries AppointmentQueries, notifications ...notificationsapp.Publisher) *Service {
	var publisher notificationsapp.Publisher
	if len(notifications) > 0 {
		publisher = notifications[0]
	}
	return &Service{repository: repository, queries: queries, notifications: publisher}
}

func (s *Service) Create(ctx context.Context, actorID, role string, businessID uuid.UUID, input domain.CreateInput) (domain.Appointment, error) {
	if strings.TrimSpace(actorID) == "" || role != "customer" {
		return domain.Appointment{}, fmt.Errorf("%w: a customer role is required", ErrForbidden)
	}
	if err := validate(&input); err != nil {
		return domain.Appointment{}, err
	}
	appointment, err := s.repository.CreateForCustomer(ctx, actorID, businessID, input)
	if err == nil && s.notifications != nil {
		s.notifications.Publish(ctx, notificationsdomain.Event{ID: "appointment_created_owner_" + appointment.ID.String(), RecipientID: appointment.BusinessOwnerID, Kind: "appointment_created", Title: "New appointment", Body: "You have a new appointment at " + appointment.BusinessName, Data: map[string]string{"type": "appointment", "appointmentId": appointment.ID.String(), "businessId": appointment.BusinessID.String()}})
	}
	return appointment, err
}
func (s *Service) publishCustomerStatus(ctx context.Context, appointment domain.Appointment) {
	if s.notifications == nil {
		return
	}
	s.notifications.Publish(ctx, notificationsdomain.Event{ID: "appointment_status_" + appointment.ID.String() + "_" + string(appointment.Status), RecipientID: appointment.CustomerID, Kind: "appointment_status_changed", Title: "Appointment updated", Body: "Your appointment at " + appointment.BusinessName + " is " + strings.ReplaceAll(string(appointment.Status), "_", " ") + ".", Data: map[string]string{"type": "appointment", "appointmentId": appointment.ID.String(), "businessId": appointment.BusinessID.String(), "status": string(appointment.Status)}})
}

func (s *Service) publishProviderCancellation(ctx context.Context, appointment domain.Appointment) {
	if s.notifications == nil {
		return
	}
	s.notifications.Publish(ctx, notificationsdomain.Event{ID: "appointment_cancelled_owner_" + appointment.ID.String(), RecipientID: appointment.BusinessOwnerID, Kind: "appointment_cancelled_by_customer", Title: "Appointment cancelled", Body: "A customer cancelled their appointment at " + appointment.BusinessName + ".", Data: map[string]string{"type": "appointment", "appointmentId": appointment.ID.String(), "businessId": appointment.BusinessID.String(), "customerId": appointment.CustomerID, "status": string(appointment.Status)}})
}
func validate(input *domain.CreateInput) error {
	if input.StaffID == uuid.Nil {
		return invalid("staff_id is required")
	}
	if _, err := time.Parse(time.DateOnly, input.AppointmentDate); err != nil {
		return invalid("appointment_date must be an ISO date")
	}
	if input.StartMinutes < 0 || input.StartMinutes >= 1440 || input.StartMinutes%30 != 0 {
		return invalid("start_minutes must be a 30-minute boundary between 0 and 1439")
	}
	if len(input.OfferingIDs) == 0 {
		return invalid("at least one offering_id is required")
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range input.OfferingIDs {
		if id == uuid.Nil || seen[id] {
			return invalid("offering_ids must be unique UUIDs")
		}
		seen[id] = true
	}
	input.CustomerName, input.CustomerEmail, input.CustomerPhone, input.PaymentMethod = strings.TrimSpace(input.CustomerName), strings.TrimSpace(input.CustomerEmail), strings.TrimSpace(input.CustomerPhone), strings.TrimSpace(input.PaymentMethod)
	if input.CustomerName == "" || input.CustomerEmail == "" || input.CustomerPhone == "" {
		return invalid("customer_name, customer_email, and customer_phone are required")
	}
	if input.PaymentMethod == "" {
		return invalid("payment_method is required")
	}
	return nil
}
func invalid(message string) error { return fmt.Errorf("%w: %s", ErrValidation, message) }
