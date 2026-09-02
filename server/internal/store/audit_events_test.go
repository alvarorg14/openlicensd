package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

func TestAuditEventCreateAndList(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	resourceID := uuid.New()
	label := "test-product"
	email := "admin@example.com"
	ip := "192.168.1.1"
	ua := "curl/8.0"
	rid := "req-123"
	meta, err := json.Marshal(map[string]string{"code": "prod-1"})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	event := store.AuditEvent{
		Action:         "product.create",
		ResourceType:   "product",
		ResourceID:     &resourceID,
		ResourceLabel:  &label,
		ActorName:      "Admin User",
		ActorEmail:     &email,
		ActorRole:      "admin",
		AuthMethod:     "session",
		ClientIP:       &ip,
		UserAgent:      &ua,
		RequestID:      &rid,
		RequestMethod:  "POST",
		RequestPath:    "/api/v1/products",
		ResponseStatus: 201,
		Metadata:       meta,
	}
	if err := st.CreateAuditEvent(ctx, event); err != nil {
		t.Fatalf("create audit event: %v", err)
	}

	tokenEvent := store.AuditEvent{
		Action:         "license.create",
		ResourceType:   "license",
		ActorName:      "ci-token",
		ActorRole:      "operator",
		AuthMethod:     "api_token",
		RequestMethod:  "POST",
		RequestPath:    "/api/v1/licenses",
		ResponseStatus: 201,
	}
	if err := st.CreateAuditEvent(ctx, tokenEvent); err != nil {
		t.Fatalf("create token audit event: %v", err)
	}

	events, total, err := st.ListAuditEvents(ctx, store.AuditEventListParams{
		ListParams: store.ListParams{
			Search: "test-product",
			Sort:   "occurred_at",
			Order:  "desc",
			Limit:  25,
			Offset: 0,
		},
		Action: "product.create",
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if total < 1 || len(events) < 1 {
		t.Fatalf("expected at least one event, total=%d len=%d", total, len(events))
	}
	if events[0].Action != "product.create" {
		t.Fatalf("action=%q want product.create", events[0].Action)
	}
	if events[0].ResourceLabel == nil || *events[0].ResourceLabel != label {
		t.Fatalf("resource_label=%v want %q", events[0].ResourceLabel, label)
	}

	filtered, _, err := st.ListAuditEvents(ctx, store.AuditEventListParams{
		ListParams: store.ListParams{Sort: "action", Order: "asc", Limit: 25},
		ResourceType: "license",
	})
	if err != nil {
		t.Fatalf("list by resource type: %v", err)
	}
	if len(filtered) < 1 {
		t.Fatalf("expected license events")
	}
}

func TestAuditEventListDateFilters(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	past := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)

	event := store.AuditEvent{
		Action:         "user.create",
		ResourceType:   "user",
		ActorName:      "Admin",
		ActorRole:      "admin",
		AuthMethod:     "session",
		RequestMethod:  "POST",
		RequestPath:    "/api/v1/users",
		ResponseStatus: 201,
	}
	if err := st.CreateAuditEvent(ctx, event); err != nil {
		t.Fatalf("create audit event: %v", err)
	}

	to := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)

	events, total, err := st.ListAuditEvents(ctx, store.AuditEventListParams{
		ListParams: store.ListParams{Sort: "occurred_at", Order: "desc", Limit: 25},
		From:       &past,
		To:         &to,
	})
	if err != nil {
		t.Fatalf("list with date filters: %v", err)
	}
	if total < 1 || len(events) < 1 {
		t.Fatalf("expected events in date range, total=%d", total)
	}
}
