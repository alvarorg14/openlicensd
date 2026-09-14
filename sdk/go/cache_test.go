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

func TestCachedValidatorCachesInvalid(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: false, Reason: ReasonNotFound})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	validator := NewCachedValidator(client, time.Minute)
	ctx := context.Background()

	result, err := validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR")
	if err != nil {
		t.Fatalf("first Validate() error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result")
	}

	result, err = validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR")
	if err != nil {
		t.Fatalf("second Validate() error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected cached invalid result")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestCachedValidatorSkipsTransportError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	validator := NewCachedValidator(client, time.Minute)
	ctx := context.Background()

	_, err = validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR")
	if err == nil {
		t.Fatal("expected transport error on first Validate()")
	}

	result, err := validator.Validate(ctx, "01234-56789-ABCDE-FGHJK-MNPQR")
	if err != nil {
		t.Fatalf("second Validate() error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid result after transport error was not cached")
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestCachedValidatorValidateProduct(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req validateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true, Product: req.Product})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	validator := NewCachedValidator(client, time.Minute)
	ctx := context.Background()
	key := "01234-56789-ABCDE-FGHJK-MNPQR"

	if _, err := validator.ValidateProduct(ctx, key, "product-a"); err != nil {
		t.Fatalf("ValidateProduct(product-a) error: %v", err)
	}
	if _, err := validator.ValidateProduct(ctx, key, "product-b"); err != nil {
		t.Fatalf("ValidateProduct(product-b) error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 for distinct products", calls)
	}

	if _, err := validator.ValidateProduct(ctx, key, "product-a"); err != nil {
		t.Fatalf("cached ValidateProduct(product-a) error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 after cache hit", calls)
	}

	validator.InvalidateProduct(key, "product-a")
	if _, err := validator.ValidateProduct(ctx, key, "product-a"); err != nil {
		t.Fatalf("ValidateProduct after invalidate error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 after invalidate", calls)
	}
}
