package application

import (
	"context"

	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

// UserRepository is the command persistence boundary required by user use cases.
type UserRepository interface {
	ProvisionFromAuthentication(ctx context.Context, input AuthenticatedUser) error
	SetRole(ctx context.Context, id string, role domain.UserRole) (domain.User, error)
	UpdateProfile(ctx context.Context, id string, input domain.UpdateProfileInput) (domain.User, error)
}

// UserQueries is the read persistence boundary required by user use cases.
type UserQueries interface {
	GetByID(ctx context.Context, id string) (domain.User, error)
}
