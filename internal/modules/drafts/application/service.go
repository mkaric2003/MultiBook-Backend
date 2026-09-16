// Package application coordinates customer-owned draft operations.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/domain"
)

var (
	ErrNotFound   = errors.New("draft not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type Service struct {
	commands Commands
	queries  Queries
}

func NewService(commands Commands, queries Queries) *Service {
	return &Service{commands: commands, queries: queries}
}

func (s *Service) GetBooking(ctx context.Context, customerID string) (domain.BookingDraft, error) {
	if err := customer(customerID); err != nil {
		return domain.BookingDraft{}, err
	}
	return s.queries.GetBooking(ctx, customerID)
}

func (s *Service) SaveBooking(ctx context.Context, customerID string, input domain.BookingInput) (domain.BookingDraft, error) {
	if err := customer(customerID); err != nil {
		return domain.BookingDraft{}, err
	}
	if err := validateBooking(input); err != nil {
		return domain.BookingDraft{}, err
	}
	if err := s.commands.UpsertBooking(ctx, customerID, input); err != nil {
		return domain.BookingDraft{}, err
	}
	return s.queries.GetBooking(ctx, customerID)
}

func (s *Service) DeleteBooking(ctx context.Context, customerID string) error {
	if err := customer(customerID); err != nil {
		return err
	}
	return s.commands.DeleteBooking(ctx, customerID)
}

func (s *Service) GetAppointment(ctx context.Context, customerID string) (domain.AppointmentDraft, error) {
	if err := customer(customerID); err != nil {
		return domain.AppointmentDraft{}, err
	}
	return s.queries.GetAppointment(ctx, customerID)
}

func (s *Service) SaveAppointment(ctx context.Context, customerID string, input domain.AppointmentInput) (domain.AppointmentDraft, error) {
	if err := customer(customerID); err != nil {
		return domain.AppointmentDraft{}, err
	}
	if err := validateAppointment(input); err != nil {
		return domain.AppointmentDraft{}, err
	}
	if err := s.commands.UpsertAppointment(ctx, customerID, input); err != nil {
		return domain.AppointmentDraft{}, err
	}
	return s.queries.GetAppointment(ctx, customerID)
}

func (s *Service) DeleteAppointment(ctx context.Context, customerID string) error {
	if err := customer(customerID); err != nil {
		return err
	}
	return s.commands.DeleteAppointment(ctx, customerID)
}

func customer(customerID string) error {
	if strings.TrimSpace(customerID) == "" {
		return fmt.Errorf("%w: authentication is required", ErrForbidden)
	}
	return nil
}

func validateBooking(input domain.BookingInput) error {
	if input.BusinessID.String() == "00000000-0000-0000-0000-000000000000" {
		return invalid("business_id is required")
	}
	checkIn, err := time.Parse(time.DateOnly, input.CheckIn)
	if err != nil {
		return invalid("check_in must be an ISO date")
	}
	checkOut, err := time.Parse(time.DateOnly, input.CheckOut)
	if err != nil || !checkOut.After(checkIn) {
		return invalid("check_out must be after check_in")
	}
	if input.Adults < 1 || input.Children < 0 || input.Infants < 0 {
		return invalid("guest counts are invalid")
	}
	return validateArray(input.SelectedExtras, "selected_extras")
}

func validateAppointment(input domain.AppointmentInput) error {
	if input.BusinessID.String() == "00000000-0000-0000-0000-000000000000" {
		return invalid("business_id is required")
	}
	if _, err := time.Parse(time.DateOnly, input.AppointmentDate); err != nil {
		return invalid("appointment_date must be an ISO date")
	}
	if input.StartMinutes != nil && (*input.StartMinutes < 0 || *input.StartMinutes >= 1440) {
		return invalid("start_minutes must be between 0 and 1439")
	}
	return validateArray(input.SelectedAddOnIDs, "selected_add_on_ids")
}

func validateArray(value json.RawMessage, field string) error {
	if len(value) == 0 {
		return nil
	}
	var list []json.RawMessage
	if err := json.Unmarshal(value, &list); err != nil {
		return invalid(field + " must be an array")
	}
	return nil
}

func invalid(message string) error { return fmt.Errorf("%w: %s", ErrValidation, message) }
