package openlicensd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCachedValidator(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	validator := NewCachedValidator(client, time.Minute)
	ctx := context.Background()

	if _, err := validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR"); err != nil {
		t.Fatalf("first Validate() error: %v", err)
	}
	if _, err := validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR"); err != nil {
		t.Fatalf("second Validate() error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}

	validator.Invalidate("01234-56789-ABCDE-FGHJK-MNPQR")
	if _, err := validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR"); err != nil {
		t.Fatalf("third Validate() error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls after invalidate = %d, want 2", calls)
	}
}
