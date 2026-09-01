package store

import (
	"context"
	"os"
	"testing"
)

func openIntegrationStore(t *testing.T) *Store {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

func TestAuditEventAppendOnlyTrigger(t *testing.T) {
	st := openIntegrationStore(t)
	ctx := context.Background()

	event := AuditEvent{
		Action:         "product.create",
		ResourceType:   "product",
		ActorName:      "Admin",
		ActorRole:      "admin",
		AuthMethod:     "session",
		RequestMethod:  "POST",
		RequestPath:    "/api/v1/products",
		ResponseStatus: 201,
	}
	if err := st.CreateAuditEvent(ctx, event); err != nil {
		t.Fatalf("create audit event: %v", err)
	}

	events, _, err := st.ListAuditEvents(ctx, AuditEventListParams{
		ListParams: ListParams{Sort: "occurred_at", Order: "desc", Limit: 1},
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	_, err = st.pool.Exec(ctx, `UPDATE audit_events SET action = $1 WHERE id = $2`, "tampered", events[0].ID)
	if err == nil {
		t.Fatal("expected update to be rejected by append-only trigger")
	}
}
