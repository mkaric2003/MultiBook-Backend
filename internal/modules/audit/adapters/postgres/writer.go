// Package postgres implements audit persistence using PostgreSQL.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
)

// Writer is the transaction-scoped Postgres audit adapter.
type Writer struct{}

func NewWriter() *Writer { return &Writer{} }

var _ application.Writer = (*Writer)(nil)

func (*Writer) Write(ctx context.Context, tx pgx.Tx, event application.Event) error {
	if event.Metadata == nil {
		_, err := tx.Exec(ctx, `INSERT INTO business_audit_events (business_id, actor_user_id, action) VALUES ($1, $2, $3)`, event.BusinessID, event.ActorUserID, event.Action)
		return err
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO business_audit_events (business_id, actor_user_id, action, metadata) VALUES ($1, $2, $3, $4::jsonb)`, event.BusinessID, event.ActorUserID, event.Action, metadata)
	return err
}
