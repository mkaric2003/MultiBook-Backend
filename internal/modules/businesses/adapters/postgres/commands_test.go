package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
)

func TestPersistedIDSeparatesExistingRowsFromDraftRows(t *testing.T) {
	t.Parallel()
	existing := uuid.New()
	if got, ok := persistedID(existing.String()); !ok || got != existing {
		t.Fatalf("persistedID() = %v, %v; want %v, true", got, ok, existing)
	}
	if got, ok := persistedID("draft-service-123"); ok || got != uuid.Nil {
		t.Fatalf("draft ID must select insert path, got %v, %v", got, ok)
	}
}

func TestBusinessAuditUsesCommandTransaction(t *testing.T) {
	t.Parallel()
	tx := &transactionStub{}
	writer := &auditWriterStub{}
	repository := &CommandRepository{audit: writer}
	businessID := uuid.New()
	if err := repository.writeAudit(context.Background(), tx, businessID, "provider", "business_updated"); err != nil {
		t.Fatal(err)
	}
	if writer.tx != tx {
		t.Fatal("audit writer did not receive the business command transaction")
	}
	if writer.event.BusinessID != businessID || writer.event.ActorUserID != "provider" || writer.event.Action != "business_updated" {
		t.Fatalf("audit event = %#v", writer.event)
	}
}

type auditWriterStub struct {
	tx    pgx.Tx
	event auditapp.Event
}

func (w *auditWriterStub) Write(_ context.Context, tx pgx.Tx, event auditapp.Event) error {
	w.tx = tx
	w.event = event
	return nil
}

type transactionStub struct{ pgx.Tx }
