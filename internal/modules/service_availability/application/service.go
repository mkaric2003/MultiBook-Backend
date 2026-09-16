// Package application coordinates provider-owned staff availability operations.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/domain"
)

var (
	ErrNotFound   = errors.New("service staff availability not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type AvailabilityRepository interface {
	CreateWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.CreateWeeklyInput) (domain.WeeklyAvailability, error)
	UpdateWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdateWeeklyInput) (domain.WeeklyAvailability, error)
	DeleteWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) error
	CreateBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.CreateBlockInput) (domain.AvailabilityBlock, error)
	UpdateBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdateBlockInput) (domain.AvailabilityBlock, error)
	DeleteBlockOwned(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type AvailabilityQueries interface {
	ListWeeklyOwned(context.Context, string, uuid.UUID, uuid.UUID) ([]domain.WeeklyAvailability, error)
	ListBlocksOwned(context.Context, string, uuid.UUID, uuid.UUID) ([]domain.AvailabilityBlock, error)
}

type Service struct {
	repository AvailabilityRepository
	queries    AvailabilityQueries
}

func NewService(repository AvailabilityRepository, queries AvailabilityQueries) *Service {
	return &Service{repository: repository, queries: queries}
}

func (s *Service) CreateWeekly(ctx context.Context, actorID, role string, businessID, staffID uuid.UUID, input domain.CreateWeeklyInput) (domain.WeeklyAvailability, error) {
	if err := provider(actorID, role); err != nil {
		return domain.WeeklyAvailability{}, err
	}
	if err := validateWeekly(input.Weekday, input.StartMinutes, input.EndMinutes); err != nil {
		return domain.WeeklyAvailability{}, err
	}
	return s.repository.CreateWeeklyOwned(ctx, actorID, businessID, staffID, input)
}
func (s *Service) ListWeekly(ctx context.Context, actorID, role string, businessID, staffID uuid.UUID) ([]domain.WeeklyAvailability, error) {
	if err := provider(actorID, role); err != nil {
		return nil, err
	}
	return s.queries.ListWeeklyOwned(ctx, actorID, businessID, staffID)
}
func (s *Service) UpdateWeekly(ctx context.Context, actorID, role string, businessID, staffID, availabilityID uuid.UUID, input domain.UpdateWeeklyInput) (domain.WeeklyAvailability, error) {
	if err := provider(actorID, role); err != nil {
		return domain.WeeklyAvailability{}, err
	}
	if input.Weekday == nil && input.StartMinutes == nil && input.EndMinutes == nil {
		return domain.WeeklyAvailability{}, validation("at least one editable field is required")
	}
	if input.Weekday != nil && (*input.Weekday < 0 || *input.Weekday > 6) {
		return domain.WeeklyAvailability{}, validation("weekday must be between 0 and 6")
	}
	if input.StartMinutes != nil && (*input.StartMinutes < 0 || *input.StartMinutes >= 1440) {
		return domain.WeeklyAvailability{}, validation("start_minutes must be between 0 and 1439")
	}
	if input.EndMinutes != nil && (*input.EndMinutes <= 0 || *input.EndMinutes > 1440) {
		return domain.WeeklyAvailability{}, validation("end_minutes must be between 1 and 1440")
	}
	if input.StartMinutes != nil && input.EndMinutes != nil && *input.StartMinutes >= *input.EndMinutes {
		return domain.WeeklyAvailability{}, validation("start_minutes must be before end_minutes")
	}
	return s.repository.UpdateWeeklyOwned(ctx, actorID, businessID, staffID, availabilityID, input)
}
func (s *Service) DeleteWeekly(ctx context.Context, actorID, role string, businessID, staffID, availabilityID uuid.UUID) error {
	if err := provider(actorID, role); err != nil {
		return err
	}
	return s.repository.DeleteWeeklyOwned(ctx, actorID, businessID, staffID, availabilityID)
}
func (s *Service) CreateBlock(ctx context.Context, actorID, role string, businessID, staffID uuid.UUID, input domain.CreateBlockInput) (domain.AvailabilityBlock, error) {
	if err := provider(actorID, role); err != nil {
		return domain.AvailabilityBlock{}, err
	}
	if err := validateBlock(input.StartAt, input.EndAt); err != nil {
		return domain.AvailabilityBlock{}, err
	}
	trimReason(&input.Reason)
	return s.repository.CreateBlockOwned(ctx, actorID, businessID, staffID, input)
}
func (s *Service) ListBlocks(ctx context.Context, actorID, role string, businessID, staffID uuid.UUID) ([]domain.AvailabilityBlock, error) {
	if err := provider(actorID, role); err != nil {
		return nil, err
	}
	return s.queries.ListBlocksOwned(ctx, actorID, businessID, staffID)
}
func (s *Service) UpdateBlock(ctx context.Context, actorID, role string, businessID, staffID, blockID uuid.UUID, input domain.UpdateBlockInput) (domain.AvailabilityBlock, error) {
	if err := provider(actorID, role); err != nil {
		return domain.AvailabilityBlock{}, err
	}
	if input.StartAt == nil && input.EndAt == nil && input.Reason == nil {
		return domain.AvailabilityBlock{}, validation("at least one editable field is required")
	}
	if input.StartAt != nil && input.StartAt.IsZero() || input.EndAt != nil && input.EndAt.IsZero() {
		return domain.AvailabilityBlock{}, validation("start_at and end_at must be valid timestamps")
	}
	if input.StartAt != nil && input.EndAt != nil {
		if err := validateBlock(*input.StartAt, *input.EndAt); err != nil {
			return domain.AvailabilityBlock{}, err
		}
	}
	trimReason(&input.Reason)
	return s.repository.UpdateBlockOwned(ctx, actorID, businessID, staffID, blockID, input)
}
func (s *Service) DeleteBlock(ctx context.Context, actorID, role string, businessID, staffID, blockID uuid.UUID) error {
	if err := provider(actorID, role); err != nil {
		return err
	}
	return s.repository.DeleteBlockOwned(ctx, actorID, businessID, staffID, blockID)
}

func provider(actorID, role string) error {
	if strings.TrimSpace(actorID) == "" || role != "provider" {
		return fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	return nil
}
func validateWeekly(weekday, start, end int16) error {
	if weekday < 0 || weekday > 6 {
		return validation("weekday must be between 0 and 6")
	}
	if start < 0 || start >= 1440 {
		return validation("start_minutes must be between 0 and 1439")
	}
	if end <= 0 || end > 1440 {
		return validation("end_minutes must be between 1 and 1440")
	}
	if start >= end {
		return validation("start_minutes must be before end_minutes")
	}
	return nil
}
func validateBlock(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return validation("start_at and end_at are required")
	}
	if !start.Before(end) {
		return validation("start_at must be before end_at")
	}
	return nil
}
func validation(message string) error { return fmt.Errorf("%w: %s", ErrValidation, message) }
func trimReason(reason **string) {
	if *reason == nil {
		return
	}
	value := strings.TrimSpace(**reason)
	*reason = &value
}
