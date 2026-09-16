// Package application contains saved payment method use cases.
package application

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/domain"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("payment method not found")
	ErrValidation = errors.New("validation failed")
)

var lastFourDigits = regexp.MustCompile(`^[0-9]{4}$`)

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

func (service *Service) List(ctx context.Context, actor Actor) ([]domain.PaymentMethod, error) {
	if err := authorizeCustomer(actor); err != nil {
		return nil, err
	}
	return service.queries.List(ctx, actor.ID)
}

func (service *Service) Create(ctx context.Context, actor Actor, input domain.CreateInput) (domain.PaymentMethod, error) {
	if err := authorizeCustomer(actor); err != nil {
		return domain.PaymentMethod{}, err
	}
	input.Last4 = strings.TrimSpace(input.Last4)
	input.HolderName = strings.TrimSpace(input.HolderName)
	currentYear := time.Now().UTC().Year()
	if !input.Brand.Valid() || !lastFourDigits.MatchString(input.Last4) || input.ExpiryMonth < 1 || input.ExpiryMonth > 12 || input.ExpiryYear < currentYear || input.ExpiryYear > currentYear+30 || input.HolderName == "" || utf8.RuneCountInString(input.HolderName) > 120 {
		return domain.PaymentMethod{}, ErrValidation
	}
	return service.commands.Create(ctx, actor.ID, input)
}

func (service *Service) SetDefault(ctx context.Context, actor Actor, paymentMethodID uuid.UUID) (domain.PaymentMethod, error) {
	if err := authorizeCustomer(actor); err != nil {
		return domain.PaymentMethod{}, err
	}
	if paymentMethodID == uuid.Nil {
		return domain.PaymentMethod{}, ErrValidation
	}
	return service.commands.SetDefault(ctx, actor.ID, paymentMethodID)
}

func (service *Service) Delete(ctx context.Context, actor Actor, paymentMethodID uuid.UUID) error {
	if err := authorizeCustomer(actor); err != nil {
		return err
	}
	if paymentMethodID == uuid.Nil {
		return ErrValidation
	}
	return service.commands.Delete(ctx, actor.ID, paymentMethodID)
}

func authorizeCustomer(actor Actor) error {
	if strings.TrimSpace(actor.ID) == "" || actor.Role != "customer" {
		return ErrForbidden
	}
	return nil
}
