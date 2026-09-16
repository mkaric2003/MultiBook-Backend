package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

const paymentMethodColumns = `id,customer_id,brand::text,last4,expiry_month,expiry_year,holder_name,is_default,created_at`

func (queries *Queries) List(ctx context.Context, customerID string) ([]domain.PaymentMethod, error) {
	rows, err := queries.pool.Query(ctx, `SELECT `+paymentMethodColumns+` FROM customer_payment_methods WHERE customer_id=$1 ORDER BY created_at DESC,id DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list payment methods: %w", err)
	}
	defer rows.Close()
	items := make([]domain.PaymentMethod, 0)
	for rows.Next() {
		method, err := scanPaymentMethod(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment method: %w", err)
		}
		items = append(items, method)
	}
	return items, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanPaymentMethod(row rowScanner) (domain.PaymentMethod, error) {
	var method domain.PaymentMethod
	var brand string
	err := row.Scan(&method.ID, &method.CustomerID, &brand, &method.Last4, &method.ExpiryMonth, &method.ExpiryYear, &method.HolderName, &method.IsDefault, &method.CreatedAt)
	method.Brand = domain.CardBrand(brand)
	return method, err
}
