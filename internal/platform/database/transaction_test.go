package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestWithinTransactionRollsBackWhenWorkFails(t *testing.T) {
	t.Parallel()
	tx := &transactionStub{}
	want := errors.New("write failed")
	err := WithinTransaction(context.Background(), beginnerStub{tx: tx}, func(pgx.Tx) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, want)
	}
	if !tx.rolledBack || tx.committed {
		t.Fatalf("transaction state = %+v, want rollback without commit", tx)
	}
}

func TestWithinTransactionCommitsSuccessfulWork(t *testing.T) {
	t.Parallel()
	tx := &transactionStub{}
	err := WithinTransaction(context.Background(), beginnerStub{tx: tx}, func(pgx.Tx) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !tx.committed {
		t.Fatal("expected commit")
	}
}

type beginnerStub struct{ tx pgx.Tx }

func (b beginnerStub) Begin(context.Context) (pgx.Tx, error) { return b.tx, nil }

type transactionStub struct {
	pgx.Tx
	committed  bool
	rolledBack bool
}

func (t *transactionStub) Commit(context.Context) error   { t.committed = true; return nil }
func (t *transactionStub) Rollback(context.Context) error { t.rolledBack = true; return nil }
