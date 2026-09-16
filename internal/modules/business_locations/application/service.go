package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_locations/domain"
	businessdomain "github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

var (
	ErrNotFound   = errors.New("business not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type LocationRepository interface {
	UpsertOwned(context.Context, string, uuid.UUID, domain.UpsertInput) (domain.Location, error)
}

type Service struct{ repository LocationRepository }

func NewService(repository LocationRepository) *Service { return &Service{repository} }
func (s *Service) Upsert(ctx context.Context, actorID, role string, businessID uuid.UUID, input domain.UpsertInput) (domain.Location, error) {
	if role != "provider" {
		return domain.Location{}, ErrForbidden
	}
	if businessdomain.TrimText(input.City) == "" || businessdomain.TrimText(input.Address) == "" || input.Latitude < -90 || input.Latitude > 90 || input.Longitude < -180 || input.Longitude > 180 {
		return domain.Location{}, fmt.Errorf("%w: valid city, address, latitude and longitude are required", ErrValidation)
	}
	input.City = businessdomain.TrimText(input.City)
	input.Address = businessdomain.TrimText(input.Address)
	input.CountryCode = businessdomain.TrimOptionalText(input.CountryCode)
	return s.repository.UpsertOwned(ctx, actorID, businessID, input)
}
