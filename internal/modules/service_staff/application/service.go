package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/domain"
	"strings"
)

var (
	ErrNotFound   = errors.New("service staff not found")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
)

type StaffRepository interface {
	CreateOwned(context.Context, string, uuid.UUID, domain.CreateInput) (domain.Staff, error)
	UpdateOwned(context.Context, string, uuid.UUID, uuid.UUID, domain.UpdateInput) (domain.Staff, error)
	ArchiveOwned(context.Context, string, uuid.UUID, uuid.UUID) error
}

type StaffQueries interface {
	ListOwned(context.Context, string, uuid.UUID) ([]domain.Staff, error)
}

type Service struct {
	repository StaffRepository
	queries    StaffQueries
}

func NewService(repository StaffRepository, queries StaffQueries) *Service {
	return &Service{repository: repository, queries: queries}
}
func (s *Service) Create(c context.Context, a, r string, b uuid.UUID, i domain.CreateInput) (domain.Staff, error) {
	if e := provider(a, r); e != nil {
		return domain.Staff{}, e
	}
	if e := validCreate(&i); e != nil {
		return domain.Staff{}, e
	}
	return s.repository.CreateOwned(c, a, b, i)
}
func (s *Service) List(c context.Context, a, r string, b uuid.UUID) ([]domain.Staff, error) {
	if e := provider(a, r); e != nil {
		return nil, e
	}
	return s.queries.ListOwned(c, a, b)
}
func (s *Service) Update(c context.Context, a, r string, b, id uuid.UUID, i domain.UpdateInput) (domain.Staff, error) {
	if e := provider(a, r); e != nil {
		return domain.Staff{}, e
	}
	if e := validUpdate(&i); e != nil {
		return domain.Staff{}, e
	}
	return s.repository.UpdateOwned(c, a, b, id, i)
}
func (s *Service) Archive(c context.Context, a, r string, b, id uuid.UUID) error {
	if e := provider(a, r); e != nil {
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
func validCreate(i *domain.CreateInput) error {
	i.Name = strings.TrimSpace(i.Name)
	if i.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if i.CommissionRate == nil {
		x := float64(100)
		i.CommissionRate = &x
	}
	if *i.CommissionRate < 0 || *i.CommissionRate > 100 {
		return fmt.Errorf("%w: commission_rate must be between 0 and 100", ErrValidation)
	}
	if i.IsActive == nil {
		x := true
		i.IsActive = &x
	}
	return nil
}
func validUpdate(i *domain.UpdateInput) error {
	if i.Name == nil && i.Title == nil && i.CommissionRate == nil && i.IsActive == nil && i.OfferingIDs == nil {
		return fmt.Errorf("%w: at least one editable field is required", ErrValidation)
	}
	if i.Name != nil {
		x := strings.TrimSpace(*i.Name)
		i.Name = &x
		if x == "" {
			return fmt.Errorf("%w: name cannot be blank", ErrValidation)
		}
	}
	if i.CommissionRate != nil && (*i.CommissionRate < 0 || *i.CommissionRate > 100) {
		return fmt.Errorf("%w: commission_rate must be between 0 and 100", ErrValidation)
	}
	return nil
}
