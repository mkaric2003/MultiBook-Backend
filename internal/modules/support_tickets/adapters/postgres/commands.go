package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/domain"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.CommandRepository = (*CommandRepository)(nil)

func (repository *CommandRepository) Create(ctx context.Context, customerID string, input domain.CreateInput) (domain.Ticket, error) {
	var ticket domain.Ticket
	err := repository.pool.QueryRow(ctx, `INSERT INTO support_tickets(
		customer_id,customer_name,customer_email,category,subject,message)
		SELECT id,COALESCE(NULLIF(trim(display_name),''),email::TEXT),email, $2,$3,$4
		FROM users WHERE id=$1 AND role='customer' AND deleted_at IS NULL AND email IS NOT NULL
		RETURNING id,customer_id,customer_name,customer_email::TEXT,category,status,subject,message,created_at,updated_at`,
		customerID, input.Category, input.Subject, input.Message,
	).Scan(&ticket.ID, &ticket.CustomerID, &ticket.CustomerName, &ticket.CustomerEmail, &ticket.Category, &ticket.Status, &ticket.Subject, &ticket.Message, &ticket.CreatedAt, &ticket.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Ticket{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("create support ticket: %w", err)
	}
	return ticket, nil
}
