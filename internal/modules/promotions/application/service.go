// Package application contains promotion use cases and ports.
package application

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("promotion not found")
	ErrValidation = errors.New("validation failed")
)

type Actor struct {
	ID   string
	Role string
}

type Service struct {
	commands CommandRepository
	queries  Queries
}

func NewService(commands CommandRepository, queries Queries) *Service {
	return &Service{commands: commands, queries: queries}
}

func (service *Service) List(ctx context.Context, actor Actor, businessID uuid.UUID) ([]domain.Promotion, error) {
	if err := authorizeProvider(actor); err != nil {
		return nil, err
	}
	return service.queries.ListOwned(ctx, actor.ID, businessID)
}

func (service *Service) Active(ctx context.Context, actor Actor, businessID uuid.UUID, promoCode *string) (*domain.Promotion, error) {
	if strings.TrimSpace(actor.ID) == "" {
		return nil, ErrForbidden
	}
	if promoCode != nil {
		normalized := strings.TrimSpace(*promoCode)
		if normalized == "" {
			promoCode = nil
		} else {
			promoCode = &normalized
		}
	}
	return service.queries.Active(ctx, businessID, promoCode)
}

func (service *Service) Create(ctx context.Context, actor Actor, businessID uuid.UUID, input domain.CreateInput) (domain.Promotion, error) {
	if err := authorizeProvider(actor); err != nil {
		return domain.Promotion{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Code != nil {
		code := strings.TrimSpace(*input.Code)
		input.Code = &code
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 160 || !input.Type.Valid() || input.Value <= 0 || input.EndsAt.Before(input.StartsAt) || input.MinimumAmount < 0 || input.MinimumNights < 0 || (input.UsageLimit != nil && *input.UsageLimit <= 0) {
		return domain.Promotion{}, ErrValidation
	}
	if (input.Type == domain.TypeCouponCode) != (input.Code != nil && *input.Code != "") {
		return domain.Promotion{}, ErrValidation
	}
	if input.Type != domain.TypeFixedAmount && input.Value > 100 {
		return domain.Promotion{}, ErrValidation
	}
	return service.commands.Create(ctx, actor.ID, businessID, input)
}

func (service *Service) SetActive(ctx context.Context, actor Actor, businessID, promotionID uuid.UUID, active bool) (domain.Promotion, error) {
	if err := authorizeProvider(actor); err != nil {
		return domain.Promotion{}, err
	}
	return service.commands.SetActive(ctx, actor.ID, businessID, promotionID, active)
}

func (service *Service) Delete(ctx context.Context, actor Actor, businessID, promotionID uuid.UUID) error {
	if err := authorizeProvider(actor); err != nil {
		return err
	}
	return service.commands.Delete(ctx, actor.ID, businessID, promotionID)
}

func authorizeProvider(actor Actor) error {
	if strings.TrimSpace(actor.ID) == "" || actor.Role != "provider" {
		return ErrForbidden
	}
	return nil
}
