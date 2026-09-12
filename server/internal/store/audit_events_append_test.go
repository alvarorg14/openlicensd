package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
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

	_, err = st.pool.Exec(ctx, `DELETE FROM audit_events WHERE id = $1`, events[0].ID)
	if err == nil {
		t.Fatal("expected delete to be rejected by append-only trigger")
	}
}

func TestDeleteAuditEventsBefore(t *testing.T) {
	st := openIntegrationStore(t)
	ctx := context.Background()

	oldID := insertAuditEventAt(t, st, ctx, time.Now().UTC().Add(-48*time.Hour))
	recentID := insertAuditEventAt(t, st, ctx, time.Now().UTC().Add(-1*time.Hour))

	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	removed, err := st.DeleteAuditEventsBefore(ctx, cutoff)
	if err != nil {
		t.Fatalf("delete audit events before: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d want 1", removed)
	}

	if auditEventExists(t, st, ctx, oldID) {
		t.Fatal("expected old audit event to be deleted")
	}
	if !auditEventExists(t, st, ctx, recentID) {
		t.Fatal("expected recent audit event to remain")
	}
}

func insertAuditEventAt(t *testing.T, st *Store, ctx context.Context, occurredAt time.Time) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := st.pool.QueryRow(ctx, `
		INSERT INTO audit_events (
			occurred_at, action, resource_type, actor_name, actor_role,
			auth_method, request_method, request_path, response_status
		)
		VALUES ($1, 'product.create', 'product', 'Admin', 'admin', 'session', 'POST', '/api/v1/products', 201)
		RETURNING id
	`, occurredAt).Scan(&id)
	if err != nil {
		t.Fatalf("insert audit event: %v", err)
	}
	return id
}

func auditEventExists(t *testing.T, st *Store, ctx context.Context, id uuid.UUID) bool {
	t.Helper()

	var exists bool
	err := st.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM audit_events WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		t.Fatalf("check audit event exists: %v", err)
	}
	return exists
}
