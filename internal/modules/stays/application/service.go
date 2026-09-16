// Package application contains stays use cases and their persistence port.
package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/domain"
)

var (
	ErrNotFound   = errors.New("stay business not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type StayRepository interface {
	UpsertOwned(context.Context, string, uuid.UUID, domain.UpsertInput) (domain.Details, error)
}

type StayQueries interface {
	GetOwned(context.Context, string, uuid.UUID) (domain.Details, error)
}

type Service struct {
	repository StayRepository
	queries    StayQueries
}

func NewService(repository StayRepository, queries StayQueries) *Service {
	return &Service{repository: repository, queries: queries}
}

func (s *Service) Upsert(ctx context.Context, actorID, role string, businessID uuid.UUID, input domain.UpsertInput) (domain.Details, error) {
	if err := requireProvider(actorID, role); err != nil {
		return domain.Details{}, err
	}
	if err := validateInput(&input); err != nil {
		return domain.Details{}, err
	}
	return s.repository.UpsertOwned(ctx, actorID, businessID, input)
}

func (s *Service) GetOwned(ctx context.Context, actorID, role string, businessID uuid.UUID) (domain.Details, error) {
	if err := requireProvider(actorID, role); err != nil {
		return domain.Details{}, err
	}
	return s.queries.GetOwned(ctx, actorID, businessID)
}

func requireProvider(actorID, role string) error {
	if strings.TrimSpace(actorID) == "" || role != "provider" {
		return fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	return nil
}

func validateInput(input *domain.UpsertInput) error {
	if input.InventoryType != domain.InventoryTypeSingleUnit && input.InventoryType != domain.InventoryTypeMultipleUnits {
		return fmt.Errorf("%w: inventory_type must be single_unit or multiple_units", ErrValidation)
	}
	if input.BasePriceMinor != nil && *input.BasePriceMinor < 0 {
		return fmt.Errorf("%w: base_price_minor cannot be negative", ErrValidation)
	}
	if input.InventoryType == domain.InventoryTypeSingleUnit && input.BasePriceMinor == nil {
		return fmt.Errorf("%w: base_price_minor is required for single_unit inventory", ErrValidation)
	}

	seen := make(map[string]struct{}, len(input.Amenities))
	amenities := make([]string, 0, len(input.Amenities))
	for _, amenity := range input.Amenities {
		amenity = strings.TrimSpace(amenity)
		if _, ok := allowedAmenities[amenity]; !ok {
			return fmt.Errorf("%w: unsupported amenity %q", ErrValidation, amenity)
		}
		if _, duplicate := seen[amenity]; !duplicate {
			seen[amenity] = struct{}{}
			amenities = append(amenities, amenity)
		}
	}
	sort.Strings(amenities)
	input.Amenities = amenities
	return nil
}

var allowedAmenities = map[string]struct{}{
	"wifi": {}, "parking": {}, "pool": {}, "spa": {}, "pet_friendly": {},
	"gym": {}, "air_conditioning": {}, "heating": {}, "kitchen": {}, "washer": {},
	"balcony": {}, "sea_view": {}, "mountain_view": {}, "workspace": {}, "elevator": {},
	"ski_in_ski_out": {}, "ski_storage": {}, "ski_rental": {}, "ski_shuttle": {},
}
