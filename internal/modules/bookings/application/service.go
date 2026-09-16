package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/domain"
	notificationsapp "github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	notificationsdomain "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

var (
	ErrNotFound   = errors.New("booking resource not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
	ErrConflict   = errors.New("stay is unavailable")
)

type Repository interface {
	CreateForCustomer(context.Context, string, uuid.UUID, domain.CreateInput) (domain.Booking, error)
	ListForCustomer(context.Context, string, domain.ListInput) (domain.Page, error)
	ListForProvider(context.Context, string, domain.ListInput) (domain.Page, error)
	CancelForCustomer(context.Context, string, uuid.UUID) (domain.Booking, error)
	UpdateStatusForProvider(context.Context, string, uuid.UUID, string) (domain.Booking, error)
	UnavailableRanges(context.Context, uuid.UUID, *uuid.UUID) ([]domain.UnavailableRange, error)
}

func (s *Service) List(ctx context.Context, actorID, role string, input domain.ListInput) (domain.Page, error) {
	if strings.TrimSpace(actorID) == "" || (role != "customer" && role != "provider") {
		return domain.Page{}, fmt.Errorf("%w: a customer or provider role is required", ErrForbidden)
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
	if input.Status != "" && input.Status != "confirmed" && input.Status != "declined" && input.Status != "cancelled" && input.Status != "completed" && input.Status != "no_show" {
		return domain.Page{}, invalid("unsupported booking status")
	}
	if role == "provider" {
		if input.BusinessID == nil {
			return domain.Page{}, invalid("business_id is required")
		}
		return s.repository.ListForProvider(ctx, actorID, input)
	}
	return s.repository.ListForCustomer(ctx, actorID, input)
}

func (s *Service) UpdateStatus(ctx context.Context, providerID, role string, bookingID uuid.UUID, status string) (domain.Booking, error) {
	if strings.TrimSpace(providerID) == "" || role != "provider" {
		return domain.Booking{}, fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	if bookingID == uuid.Nil {
		return domain.Booking{}, invalid("booking_id is required")
	}
	if status != "declined" && status != "completed" && status != "no_show" {
		return domain.Booking{}, invalid("unsupported booking status")
	}
	booking, err := s.repository.UpdateStatusForProvider(ctx, providerID, bookingID, status)
	if err == nil {
		s.publishCustomerStatus(ctx, booking)
	}
	return booking, err
}

func (s *Service) Cancel(ctx context.Context, customerID, role string, bookingID uuid.UUID) (domain.Booking, error) {
	if strings.TrimSpace(customerID) == "" || role != "customer" {
		return domain.Booking{}, fmt.Errorf("%w: a customer role is required", ErrForbidden)
	}
	if bookingID == uuid.Nil {
		return domain.Booking{}, invalid("booking_id is required")
	}
	booking, err := s.repository.CancelForCustomer(ctx, customerID, bookingID)
	if err == nil {
		s.publishProviderCancellation(ctx, booking)
	}
	return booking, err
}

func (s *Service) Availability(ctx context.Context, customerID, role string, businessID uuid.UUID, roomTypeID *uuid.UUID) ([]domain.UnavailableRange, error) {
	if strings.TrimSpace(customerID) == "" || role != "customer" {
		return nil, fmt.Errorf("%w: a customer role is required", ErrForbidden)
	}
	if businessID == uuid.Nil {
		return nil, invalid("business_id is required")
	}
	return s.repository.UnavailableRanges(ctx, businessID, roomTypeID)
}

type Service struct {
	repository    Repository
	notifications notificationsapp.Publisher
}

func NewService(repository Repository, notifications ...notificationsapp.Publisher) *Service {
	var publisher notificationsapp.Publisher
	if len(notifications) > 0 {
		publisher = notifications[0]
	}
	return &Service{repository: repository, notifications: publisher}
}
func (s *Service) Create(ctx context.Context, customerID, role string, businessID uuid.UUID, input domain.CreateInput) (domain.Booking, error) {
	if strings.TrimSpace(customerID) == "" || role != "customer" {
		return domain.Booking{}, fmt.Errorf("%w: a customer role is required", ErrForbidden)
	}
	if businessID == uuid.Nil {
		return domain.Booking{}, invalid("business_id is required")
	}
	if _, err := time.Parse(time.DateOnly, input.CheckIn); err != nil {
		return domain.Booking{}, invalid("check_in must be an ISO date")
	}
	checkOut, err := time.Parse(time.DateOnly, input.CheckOut)
	if err != nil {
		return domain.Booking{}, invalid("check_out must be an ISO date")
	}
	checkIn, _ := time.Parse(time.DateOnly, input.CheckIn)
	if !checkOut.After(checkIn) {
		return domain.Booking{}, invalid("check_out must be after check_in")
	}
	if input.Adults < 1 || input.Children < 0 || input.Infants < 0 {
		return domain.Booking{}, invalid("guest counts are invalid")
	}
	input.CustomerName, input.CustomerEmail, input.PaymentMethod = strings.TrimSpace(input.CustomerName), strings.TrimSpace(input.CustomerEmail), strings.TrimSpace(input.PaymentMethod)
	if input.CustomerName == "" || input.CustomerEmail == "" || input.PaymentMethod == "" {
		return domain.Booking{}, invalid("customer_name, customer_email, and payment_method are required")
	}
	seen := map[string]bool{}
	for _, extra := range input.SelectedExtras {
		extra.Type = strings.TrimSpace(extra.Type)
		if extra.Type == "" || seen[extra.Type] {
			return domain.Booking{}, invalid("selected_extras must have unique types")
		}
		seen[extra.Type] = true
	}
	booking, err := s.repository.CreateForCustomer(ctx, customerID, businessID, input)
	if err == nil && s.notifications != nil {
		s.notifications.Publish(ctx, notificationsdomain.Event{ID: "booking_created_owner_" + booking.ID.String(), RecipientID: booking.BusinessOwnerID, Kind: "booking_created", Title: "New booking", Body: "You have a new booking at " + booking.BusinessName, Data: map[string]string{"type": "booking", "bookingId": booking.ID.String(), "businessId": booking.BusinessID.String()}})
	}
	return booking, err
}

func (s *Service) publishCustomerStatus(ctx context.Context, booking domain.Booking) {
	if s.notifications == nil {
		return
	}
	s.notifications.Publish(ctx, notificationsdomain.Event{ID: "booking_status_" + booking.ID.String() + "_" + booking.Status, RecipientID: booking.CustomerID, Kind: "booking_status_changed", Title: "Booking updated", Body: "Your booking at " + booking.BusinessName + " is " + strings.ReplaceAll(booking.Status, "_", " ") + ".", Data: map[string]string{"type": "booking", "bookingId": booking.ID.String(), "businessId": booking.BusinessID.String(), "status": booking.Status}})
}

func (s *Service) publishProviderCancellation(ctx context.Context, booking domain.Booking) {
	if s.notifications == nil {
		return
	}
	s.notifications.Publish(ctx, notificationsdomain.Event{ID: "booking_cancelled_owner_" + booking.ID.String(), RecipientID: booking.BusinessOwnerID, Kind: "booking_cancelled_by_customer", Title: "Booking cancelled", Body: "A customer cancelled their booking at " + booking.BusinessName + ".", Data: map[string]string{"type": "booking", "bookingId": booking.ID.String(), "businessId": booking.BusinessID.String(), "customerId": booking.CustomerID, "status": booking.Status}})
}
func invalid(message string) error { return fmt.Errorf("%w: %s", ErrValidation, message) }
