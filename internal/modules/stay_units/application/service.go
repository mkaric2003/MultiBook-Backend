// Package application contains stay unit type use cases and their port.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/domain"
)

var (
	ErrNotFound   = errors.New("stay unit type not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type UnitTypeRepository interface {
	CreateOwned(context.Context, string, uuid.UUID, domain.CreateInput) (domain.UnitType, error)
	UpdateOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.UpdateInput) (domain.UnitType, error)
	ArchiveOwned(context.Context, string, uuid.UUID, uuid.UUID) error
}

type UnitTypeQueries interface {
	ListOwned(context.Context, string, uuid.UUID) ([]domain.UnitType, error)
}

type Service struct {
	repository UnitTypeRepository
	queries    UnitTypeQueries
}

func NewService(repository UnitTypeRepository, queries UnitTypeQueries) *Service {
	return &Service{repository: repository, queries: queries}
}

func (s *Service) Create(ctx context.Context, actorID, role string, businessID uuid.UUID, input domain.CreateInput) (domain.UnitType, error) {
	if err := requireProvider(actorID, role); err != nil {
		return domain.UnitType{}, err
	}
	if err := validateCreate(&input); err != nil {
		return domain.UnitType{}, err
	}
	return s.repository.CreateOwned(ctx, actorID, businessID, input)
}

func (s *Service) List(ctx context.Context, actorID, role string, businessID uuid.UUID) ([]domain.UnitType, error) {
	if err := requireProvider(actorID, role); err != nil {
		return nil, err
	}
	return s.queries.ListOwned(ctx, actorID, businessID)
}

func (s *Service) Update(ctx context.Context, actorID, role string, businessID, unitTypeID uuid.UUID, input domain.UpdateInput) (domain.UnitType, error) {
	if err := requireProvider(actorID, role); err != nil {
		return domain.UnitType{}, err
	}
	if err := validateUpdate(&input); err != nil {
		return domain.UnitType{}, err
	}
	return s.repository.UpdateOwned(ctx, actorID, businessID, unitTypeID, input)
}

func (s *Service) Archive(ctx context.Context, actorID, role string, businessID, unitTypeID uuid.UUID) error {
	if err := requireProvider(actorID, role); err != nil {
		return err
	}
	return s.repository.ArchiveOwned(ctx, actorID, businessID, unitTypeID)
}

func requireProvider(actorID, role string) error {
	if strings.TrimSpace(actorID) == "" || role != "provider" {
		return fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	return nil
}

func validateCreate(input *domain.CreateInput) error {
	if input.IsActive == nil {
		active := true
		input.IsActive = &active
	}
	if err := validateValues(input.Name, input.MaxGuests, input.SizeSquareMeters, input.PricePerNightMinor, input.Quantity); err != nil {
		return err
	}
	input.Name = strings.TrimSpace(input.Name)
	return nil
}

func validateUpdate(input *domain.UpdateInput) error {
	if input.Name == nil && input.MaxGuests == nil && input.SizeSquareMeters == nil && input.PricePerNightMinor == nil && input.Quantity == nil && input.IsActive == nil {
		return fmt.Errorf("%w: at least one editable field is required", ErrValidation)
	}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		input.Name = &trimmed
		if *input.Name == "" {
			return fmt.Errorf("%w: name cannot be blank", ErrValidation)
		}
	}
	if input.MaxGuests != nil && *input.MaxGuests <= 0 {
		return fmt.Errorf("%w: max_guests must be greater than zero", ErrValidation)
	}
	if input.SizeSquareMeters != nil && *input.SizeSquareMeters < 0 {
		return fmt.Errorf("%w: size_square_meters cannot be negative", ErrValidation)
	}
	if input.PricePerNightMinor != nil && *input.PricePerNightMinor < 0 {
		return fmt.Errorf("%w: price_per_night_minor cannot be negative", ErrValidation)
	}
	if input.Quantity != nil && *input.Quantity <= 0 {
		return fmt.Errorf("%w: quantity must be greater than zero", ErrValidation)
	}
	return nil
}

func validateValues(name string, maxGuests int16, sizeSquareMeters int, pricePerNightMinor int64, quantity int16) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if maxGuests <= 0 {
		return fmt.Errorf("%w: max_guests must be greater than zero", ErrValidation)
	}
	if sizeSquareMeters < 0 {
		return fmt.Errorf("%w: size_square_meters cannot be negative", ErrValidation)
	}
	if pricePerNightMinor < 0 {
		return fmt.Errorf("%w: price_per_night_minor cannot be negative", ErrValidation)
	}
	if quantity <= 0 {
		return fmt.Errorf("%w: quantity must be greater than zero", ErrValidation)
	}
	return nil
}
