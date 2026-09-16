package application

import (
	"context"

	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/domain"
)

type CommandRepository interface {
	Create(ctx context.Context, customerID string, input domain.CreateInput) (domain.Ticket, error)
}

type Queries interface {
	List(ctx context.Context, customerID string, limit, offset int) ([]domain.Ticket, error)
}
