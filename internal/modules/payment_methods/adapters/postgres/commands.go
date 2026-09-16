package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.CommandRepository = (*CommandRepository)(nil)

func (repository *CommandRepository) Create(ctx context.Context, customerID string, input domain.CreateInput) (domain.PaymentMethod, error) {
	var method domain.PaymentMethod
	err := database.WithinTransaction(ctx, repository.pool, func(tx pgx.Tx) error {
		if err := lockCustomer(ctx, tx, customerID); err != nil {
			return err
		}
		var err error
		method, err = scanPaymentMethod(tx.QueryRow(ctx, `INSERT INTO customer_payment_methods(customer_id,brand,last4,expiry_month,expiry_year,holder_name,is_default)
			VALUES($1,$2::saved_card_brand,$3,$4,$5,$6,NOT EXISTS(SELECT 1 FROM customer_payment_methods WHERE customer_id=$1))
			RETURNING `+paymentMethodColumns, customerID, input.Brand, input.Last4, input.ExpiryMonth, input.ExpiryYear, input.HolderName))
		if err != nil {
			return fmt.Errorf("create payment method: %w", err)
		}
		return nil
	})
	return method, err
}

func (repository *CommandRepository) SetDefault(ctx context.Context, customerID string, paymentMethodID uuid.UUID) (domain.PaymentMethod, error) {
	var method domain.PaymentMethod
	err := database.WithinTransaction(ctx, repository.pool, func(tx pgx.Tx) error {
		if err := lockCustomer(ctx, tx, customerID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE customer_payment_methods SET is_default=FALSE WHERE customer_id=$1 AND is_default`, customerID); err != nil {
			return fmt.Errorf("clear default payment method: %w", err)
		}
		var err error
		method, err = scanPaymentMethod(tx.QueryRow(ctx, `UPDATE customer_payment_methods SET is_default=TRUE WHERE id=$1 AND customer_id=$2 RETURNING `+paymentMethodColumns, paymentMethodID, customerID))
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("set default payment method: %w", err)
		}
		return nil
	})
	return method, err
}

func lockCustomer(ctx context.Context, tx pgx.Tx, customerID string) error {
	var customer bool
	err := tx.QueryRow(ctx, `SELECT COALESCE(role='customer',FALSE) FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, customerID).Scan(&customer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrForbidden
		}
		return fmt.Errorf("lock payment method customer: %w", err)
	}
	if !customer {
		return application.ErrForbidden
	}
	return nil
}

func (repository *CommandRepository) Delete(ctx context.Context, customerID string, paymentMethodID uuid.UUID) error {
	command, err := repository.pool.Exec(ctx, `DELETE FROM customer_payment_methods WHERE id=$1 AND customer_id=$2`, paymentMethodID, customerID)
	if err != nil {
		return fmt.Errorf("delete payment method: %w", err)
	}
	if command.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}
