// Package application defines transaction-scoped audit persistence ports.
package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Event is a business-scoped audit record. The port is intentionally generic
// so future booking, appointment and payment commands can record their
// business-related actions without depending on a concrete Postgres adapter.
type Event struct {
	BusinessID  uuid.UUID
	ActorUserID string
	Action      string
	Metadata    map[string]any
}

// Writer records an audit event through the caller's transaction. Callers own
// the transaction boundary, keeping the command and its audit row atomic.
type Writer interface {
	Write(context.Context, pgx.Tx, Event) error
}
