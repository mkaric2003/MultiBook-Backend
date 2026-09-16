package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/domain"
	"strings"
)

var (
	ErrNotFound   = errors.New("service offering not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type OfferingRepository interface {
	CreateOwned(context.Context, string, uuid.UUID, domain.CreateInput) (domain.Offering, error)
	UpdateOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.UpdateInput) (domain.Offering, error)
	ArchiveOwned(context.Context, string, uuid.UUID, uuid.UUID) error
}

type OfferingQueries interface {
	ListOwned(context.Context, string, uuid.UUID) ([]domain.Offering, error)
}

type Service struct {
	repository OfferingRepository
	queries    OfferingQueries
}

func NewService(repository OfferingRepository, queries OfferingQueries) *Service {
	return &Service{repository: repository, queries: queries}
}
func (s *Service) Create(c context.Context, a, role string, b uuid.UUID, i domain.CreateInput) (domain.Offering, error) {
	if e := provider(a, role); e != nil {
		return domain.Offering{}, e
	}
	if e := create(&i); e != nil {
		return domain.Offering{}, e
	}
	return s.repository.CreateOwned(c, a, b, i)
}
func (s *Service) List(c context.Context, a, role string, b uuid.UUID) ([]domain.Offering, error) {
	if e := provider(a, role); e != nil {
		return nil, e
	}
	return s.queries.ListOwned(c, a, b)
}
func (s *Service) Update(c context.Context, a, role string, b, id uuid.UUID, i domain.UpdateInput) (domain.Offering, error) {
	if e := provider(a, role); e != nil {
		return domain.Offering{}, e
	}
	if e := update(&i); e != nil {
		return domain.Offering{}, e
	}
	return s.repository.UpdateOwned(c, a, b, id, i)
}
func (s *Service) Archive(c context.Context, a, role string, b, id uuid.UUID) error {
	if e := provider(a, role); e != nil {
		return e
	}
	return s.repository.ArchiveOwned(c, a, b, id)
}
func provider(a, r string) error {
	if strings.TrimSpace(a) == "" || r != "provider" {
		return fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	return nil
}
func create(i *domain.CreateInput) error {
	if i.IsActive == nil {
		x := true
		i.IsActive = &x
	}
	i.Name = strings.TrimSpace(i.Name)
	if i.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if i.DurationMinutes < 5 || i.DurationMinutes > 1440 {
		return fmt.Errorf("%w: duration_minutes must be between 5 and 1440", ErrValidation)
	}
	if i.PriceMinor < 0 {
		return fmt.Errorf("%w: price_minor cannot be negative", ErrValidation)
	}
	if i.Description != nil {
		x := strings.TrimSpace(*i.Description)
		i.Description = &x
	}
	return nil
}
func update(i *domain.UpdateInput) error {
	if i.Name == nil && i.Description == nil && i.DurationMinutes == nil && i.PriceMinor == nil && i.IsActive == nil {
		return fmt.Errorf("%w: at least one editable field is required", ErrValidation)
	}
	if i.Name != nil {
		x := strings.TrimSpace(*i.Name)
		i.Name = &x
		if x == "" {
			return fmt.Errorf("%w: name cannot be blank", ErrValidation)
		}
	}
	if i.DurationMinutes != nil && (*i.DurationMinutes < 5 || *i.DurationMinutes > 1440) {
		return fmt.Errorf("%w: duration_minutes must be between 5 and 1440", ErrValidation)
	}
	if i.PriceMinor != nil && *i.PriceMinor < 0 {
		return fmt.Errorf("%w: price_minor cannot be negative", ErrValidation)
	}
	return nil
}
