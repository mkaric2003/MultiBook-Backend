package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/domain"
)

var (
	ErrNotFound   = errors.New("service business not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type ServiceRepository interface {
	UpsertOwned(context.Context, string, uuid.UUID, domain.UpsertInput) (domain.Details, error)
}

type ServiceQueries interface {
	GetOwned(context.Context, string, uuid.UUID) (domain.Details, error)
}

type Service struct {
	repository ServiceRepository
	queries    ServiceQueries
}

func NewService(repository ServiceRepository, queries ServiceQueries) *Service {
	return &Service{repository: repository, queries: queries}
}
func (s *Service) Upsert(ctx context.Context, actorID, role string, businessID uuid.UUID, input domain.UpsertInput) (domain.Details, error) {
	if err := provider(actorID, role); err != nil {
		return domain.Details{}, err
	}
	input.TimeZone = strings.TrimSpace(input.TimeZone)
	if input.TimeZone == "" {
		return domain.Details{}, fmt.Errorf("%w: time_zone is required", ErrValidation)
	}
	if _, err := time.LoadLocation(input.TimeZone); err != nil {
		return domain.Details{}, fmt.Errorf("%w: time_zone must be an IANA time zone", ErrValidation)
	}
	return s.repository.UpsertOwned(ctx, actorID, businessID, input)
}
func (s *Service) Get(ctx context.Context, actorID, role string, businessID uuid.UUID) (domain.Details, error) {
	if err := provider(actorID, role); err != nil {
		return domain.Details{}, err
	}
	return s.queries.GetOwned(ctx, actorID, businessID)
}
func provider(actorID, role string) error {
	if strings.TrimSpace(actorID) == "" || role != "provider" {
		return fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	return nil
}
