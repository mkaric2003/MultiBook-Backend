// Package application contains user use cases and their ports.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrValidation   = errors.New("validation failed")
)

type Service struct {
	repository UserRepository
	queries    UserQueries
}

func NewService(repository UserRepository, queries UserQueries) *Service {
	return &Service{repository: repository, queries: queries}
}

// AuthenticatedUser is authentication-provider-neutral data used to provision
// the corresponding MultiBook user record.
type AuthenticatedUser struct {
	ID          string
	Email       string
	DisplayName string
}

func (s *Service) CurrentUser(ctx context.Context, authenticatedUser AuthenticatedUser) (domain.User, error) {
	if strings.TrimSpace(authenticatedUser.ID) == "" {
		return domain.User{}, fmt.Errorf("missing authenticated user")
	}
	if err := s.repository.ProvisionFromAuthentication(ctx, authenticatedUser); err != nil {
		return domain.User{}, err
	}
	return s.queries.GetByID(ctx, authenticatedUser.ID)
}

func (s *Service) SetRole(ctx context.Context, id string, role domain.UserRole) (domain.User, error) {
	if role != domain.UserRoleCustomer && role != domain.UserRoleProvider {
		return domain.User{}, fmt.Errorf("%w: role must be customer or provider", ErrValidation)
	}
	return s.repository.SetRole(ctx, id, role)
}

func (s *Service) UpdateProfile(ctx context.Context, id string, input domain.UpdateProfileInput) (domain.User, error) {
	if input.DateOfBirth != nil {
		if _, err := time.Parse("2006-01-02", *input.DateOfBirth); err != nil {
			return domain.User{}, fmt.Errorf("%w: date_of_birth must use YYYY-MM-DD", ErrValidation)
		}
	}
	if input.BusinessCurrency != nil && !validCurrency(*input.BusinessCurrency) {
		return domain.User{}, fmt.Errorf("%w: unsupported business_currency", ErrValidation)
	}
	return s.repository.UpdateProfile(ctx, id, input)
}

func validCurrency(value string) bool {
	switch value {
	case "BAM", "USD", "EUR", "CHF", "GBP":
		return true
	}
	return false
}
