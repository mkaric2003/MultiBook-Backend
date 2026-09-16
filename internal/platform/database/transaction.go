package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// TransactionBeginner is implemented by pgxpool.Pool and lets application
// command orchestration depend on one small, typed pgx transaction boundary.
type TransactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// WithinTransaction executes work atomically. A callback error rolls the
// transaction back; successful work commits. It intentionally exposes pgx.Tx
// rather than introducing a repository framework or ORM abstraction.
func WithinTransaction(ctx context.Context, beginner TransactionBeginner, work func(pgx.Tx) error) error {
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := work(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
