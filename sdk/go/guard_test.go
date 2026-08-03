package openlicensd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGuardValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true, Product: "acme-widget"})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	guard, err := NewGuard(ctx, client, "TEST-KEY", WithInterval(50*time.Millisecond), WithOfflineGrace(time.Minute))
	if err != nil {
		t.Fatalf("NewGuard() error: %v", err)
	}
	defer guard.Stop()

	if !guard.Valid() {
		t.Fatal("expected guard to be valid")
	}
	if guard.Last().Product != "acme-widget" {
		t.Fatalf("last product = %q", guard.Last().Product)
	}
}

func TestGuardOfflineGrace(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
			return
		}
		http.Error(w, "connection reset", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	guard, err := NewGuard(ctx, client, "TEST-KEY", WithInterval(20*time.Millisecond), WithOfflineGrace(time.Minute))
	if err != nil {
		t.Fatalf("NewGuard() error: %v", err)
	}
	defer guard.Stop()

	time.Sleep(80 * time.Millisecond)

	if !guard.Valid() {
		t.Fatal("expected guard to remain valid within offline grace")
	}
}
