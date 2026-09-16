package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (queries *Queries) List(ctx context.Context, customerID string, limit, offset int) ([]domain.Ticket, error) {
	rows, err := queries.pool.Query(ctx, `SELECT id,customer_id,customer_name,customer_email::TEXT,
		category,status,subject,message,created_at,updated_at
		FROM support_tickets WHERE customer_id=$1
		ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, customerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list support tickets: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Ticket, 0)
	for rows.Next() {
		var ticket domain.Ticket
		if err := rows.Scan(&ticket.ID, &ticket.CustomerID, &ticket.CustomerName, &ticket.CustomerEmail, &ticket.Category, &ticket.Status, &ticket.Subject, &ticket.Message, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan support ticket: %w", err)
		}
		items = append(items, ticket)
	}
	return items, rows.Err()
}
