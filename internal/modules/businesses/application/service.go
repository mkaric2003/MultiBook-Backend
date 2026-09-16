// Package application contains business use cases and their ports.
package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

var (
	ErrBusinessNotFound = errors.New("business not found")
	ErrForbidden        = errors.New("forbidden")
	ErrValidation       = errors.New("validation failed")
	ErrBusinessArchived = errors.New("archived business cannot be edited")
	ErrTypeImmutable    = errors.New("business type cannot be changed")
)

type Actor struct {
	ID       string
	Role     string
	Currency string
}

type Service struct {
	repository BusinessRepository
	queries    BusinessQueries
}

func NewService(repository BusinessRepository, queries BusinessQueries) *Service {
	return &Service{repository: repository, queries: queries}
}

func (s *Service) Create(ctx context.Context, actor Actor, input domain.CreateInput) (CreatedBusiness, error) {
	if err := requireProvider(actor); err != nil {
		return CreatedBusiness{}, err
	}
	if err := validateCreate(input); err != nil {
		return CreatedBusiness{}, err
	}
	if !validCurrency(actor.Currency) {
		return CreatedBusiness{}, fmt.Errorf("%w: user business currency is invalid", ErrValidation)
	}
	input = prepareCreateInput(input)
	persisted, err := s.repository.Create(ctx, actor, input)
	if err != nil {
		return CreatedBusiness{}, err
	}
	return CreatedBusiness{Business: persisted.Business, Input: withAggregateIDs(input, persisted.AggregateIDs)}, nil
}

func (s *Service) Replace(ctx context.Context, actor Actor, id uuid.UUID, input domain.CreateInput) (CreatedBusiness, error) {
	if err := requireProvider(actor); err != nil {
		return CreatedBusiness{}, err
	}
	if err := validateCreate(input); err != nil {
		return CreatedBusiness{}, err
	}
	if err := s.requireEditableOwned(ctx, actor, id, input.Type); err != nil {
		return CreatedBusiness{}, err
	}
	input = prepareCreateInput(input)
	persisted, err := s.repository.ReplaceOwned(ctx, actor, id, input)
	if err != nil {
		return CreatedBusiness{}, err
	}
	return CreatedBusiness{Business: persisted.Business, Input: withAggregateIDs(input, persisted.AggregateIDs)}, nil
}

func (s *Service) ListOwned(ctx context.Context, actor Actor) ([]OwnedSummary, error) {
	if err := requireProvider(actor); err != nil {
		return nil, err
	}
	return s.queries.ListOwned(ctx, actor.ID)
}

func (s *Service) GetOwned(ctx context.Context, actor Actor, id uuid.UUID) (OwnedDetail, error) {
	if err := requireProvider(actor); err != nil {
		return OwnedDetail{}, err
	}
	return s.queries.GetOwnedDetail(ctx, actor.ID, id)
}

func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, input domain.UpdateInput) (domain.Business, error) {
	if err := requireProvider(actor); err != nil {
		return domain.Business{}, err
	}
	if err := validateUpdate(input); err != nil {
		return domain.Business{}, err
	}
	if err := s.requireEditableOwned(ctx, actor, id, ""); err != nil {
		return domain.Business{}, err
	}
	input.Name = domain.TrimOptionalText(input.Name)
	input.CategoryID = domain.TrimOptionalText(input.CategoryID)
	input.ShortDescription = domain.TrimOptionalText(input.ShortDescription)
	return s.repository.UpdateOwned(ctx, actor, id, input)
}

// requireEditableOwned is intentionally called before a command adapter is
// reached. In particular, ReplaceOwned must not start reconciling nested rows
// until ownership and lifecycle are known to be valid.
func (s *Service) requireEditableOwned(ctx context.Context, actor Actor, id uuid.UUID, requestedType domain.BusinessType) error {
	business, err := s.queries.GetOwnedDetail(ctx, actor.ID, id)
	if err != nil {
		return err
	}
	if business.Status == domain.BusinessStatusArchived {
		return fmt.Errorf("%w: %w", ErrValidation, ErrBusinessArchived)
	}
	if requestedType != "" && business.Type != requestedType {
		return fmt.Errorf("%w: %w", ErrValidation, ErrTypeImmutable)
	}
	return nil
}

func (s *Service) Archive(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireProvider(actor); err != nil {
		return err
	}
	return s.repository.ArchiveOwned(ctx, actor, id)
}

func (s *Service) Select(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireProvider(actor); err != nil {
		return err
	}
	return s.repository.SetSelectedBusiness(ctx, actor, id)
}

func requireProvider(actor Actor) error {
	if domain.TrimText(actor.ID) == "" {
		return fmt.Errorf("%w: authenticated user is required", ErrForbidden)
	}
	if actor.Role != "provider" {
		return fmt.Errorf("%w: a provider role is required", ErrForbidden)
	}
	return nil
}

func validateCreate(input domain.CreateInput) error {
	if input.Type != domain.BusinessTypeStay && input.Type != domain.BusinessTypeService {
		return fmt.Errorf("%w: type must be stay or service", ErrValidation)
	}
	if domain.TrimText(input.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if domain.TrimText(input.CategoryID) == "" {
		return fmt.Errorf("%w: category_id is required", ErrValidation)
	}
	if domain.TrimText(input.Location.City) == "" || domain.TrimText(input.Location.Address) == "" || input.Location.Latitude < -90 || input.Location.Latitude > 90 || input.Location.Longitude < -180 || input.Location.Longitude > 180 {
		return fmt.Errorf("%w: a valid location is required", ErrValidation)
	}
	if len(input.Media) > 9 {
		return fmt.Errorf("%w: at most 9 media items are allowed", ErrValidation)
	}
	if input.Type == domain.BusinessTypeStay && (input.Stay == nil || input.Service != nil) {
		return fmt.Errorf("%w: stay details are required", ErrValidation)
	}
	if input.Type == domain.BusinessTypeService && (input.Service == nil || input.Stay != nil) {
		return fmt.Errorf("%w: service details are required", ErrValidation)
	}
	if input.Stay != nil && input.Stay.InventoryType != "single_unit" && input.Stay.InventoryType != "multiple_units" {
		return fmt.Errorf("%w: invalid stay inventory_type", ErrValidation)
	}
	if input.Service != nil && (domain.TrimText(input.Service.TimeZone) == "" || len(input.Service.Offerings) == 0 || len(input.Service.Staff) == 0) {
		return fmt.Errorf("%w: time_zone, offerings, and staff are required", ErrValidation)
	}
	return nil
}

func validateUpdate(input domain.UpdateInput) error {
	if input.Name == nil && input.CategoryID == nil && input.ShortDescription == nil && input.Status == nil {
		return fmt.Errorf("%w: at least one editable field is required", ErrValidation)
	}
	if input.Name != nil && domain.TrimText(*input.Name) == "" {
		return fmt.Errorf("%w: name cannot be blank", ErrValidation)
	}
	if input.CategoryID != nil && domain.TrimText(*input.CategoryID) == "" {
		return fmt.Errorf("%w: category_id cannot be blank", ErrValidation)
	}
	if input.Status != nil && *input.Status != domain.BusinessStatusActive && *input.Status != domain.BusinessStatusInactive {
		return fmt.Errorf("%w: status must be active or inactive", ErrValidation)
	}
	return nil
}

func validCurrency(value string) bool {
	switch value {
	case "BAM", "USD", "EUR", "CHF", "GBP":
		return true
	}
	return false
}

func prepareCreateInput(input domain.CreateInput) domain.CreateInput {
	input.Name = domain.TrimText(input.Name)
	input.CategoryID = domain.TrimText(input.CategoryID)
	input.ShortDescription = domain.TrimOptionalText(input.ShortDescription)
	input.Location.City = domain.TrimText(input.Location.City)
	input.Location.Address = domain.TrimText(input.Location.Address)
	input.Location.CountryCode = domain.TrimOptionalText(input.Location.CountryCode)
	return input
}

func withAggregateIDs(input domain.CreateInput, ids AggregateIDMapping) domain.CreateInput {
	result := input
	if input.Stay != nil {
		stay := *input.Stay
		stay.UnitTypes = append([]domain.UnitType(nil), input.Stay.UnitTypes...)
		for index := range stay.UnitTypes {
			if index < len(ids.StayUnitTypeIDs) {
				stay.UnitTypes[index].ID = ids.StayUnitTypeIDs[index]
			}
		}
		result.Stay = &stay
	}
	if input.Service != nil {
		service := *input.Service
		service.Offerings = append([]domain.Offering(nil), input.Service.Offerings...)
		service.Staff = append([]domain.Staff(nil), input.Service.Staff...)
		for index := range service.Offerings {
			if index < len(ids.ServiceOfferingIDs) {
				service.Offerings[index].ID = ids.ServiceOfferingIDs[index]
			}
		}
		for staffIndex := range service.Staff {
			staff := &service.Staff[staffIndex]
			staff.WeeklyAvailability = append([]domain.WeeklyAvailability(nil), input.Service.Staff[staffIndex].WeeklyAvailability...)
			if staffIndex < len(ids.ServiceStaffIDs) {
				staff.ID = ids.ServiceStaffIDs[staffIndex]
			}
			if staffIndex < len(ids.ServiceStaffAvailabilityIDs) {
				for availabilityIndex := range staff.WeeklyAvailability {
					if availabilityIndex < len(ids.ServiceStaffAvailabilityIDs[staffIndex]) {
						staff.WeeklyAvailability[availabilityIndex].ID = ids.ServiceStaffAvailabilityIDs[staffIndex][availabilityIndex]
					}
				}
			}
		}
		result.Service = &service
	}
	return result
}
