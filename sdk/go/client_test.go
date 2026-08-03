package openlicensd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		product string
		opts    []Option
		wantErr error
	}{
		{
			name:    "valid",
			baseURL: "https://licenses.example.com",
			product: "acme-widget",
		},
		{
			name:    "trailing slash stripped",
			baseURL: "https://licenses.example.com/",
			product: "acme-widget",
		},
		{
			name:    "path prefix",
			baseURL: "https://example.com/licenses",
			product: "acme-widget",
		},
		{
			name:    "missing product",
			baseURL: "https://licenses.example.com",
			product: "",
			wantErr: ErrMissingProduct,
		},
		{
			name:    "any product",
			baseURL: "https://licenses.example.com",
			product: "",
			opts:    []Option{WithAnyProduct()},
		},
		{
			name:    "empty url",
			baseURL: "",
			product: "acme-widget",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "bad scheme",
			baseURL: "ftp://licenses.example.com",
			product: "acme-widget",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.baseURL, tt.product, tt.opts...)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("New() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}
		})
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv(envURL, "https://licenses.example.com")
	t.Setenv(envProduct, "acme-widget")

	client, err := NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv() error: %v", err)
	}
	if client.product != "acme-widget" {
		t.Fatalf("product = %q, want acme-widget", client.product)
	}
}

func TestNewFromEnvMissing(t *testing.T) {
	t.Setenv(envURL, "")
	t.Setenv(envProduct, "")

	_, err := NewFromEnv()
	if !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("NewFromEnv() error = %v, want ErrMissingEnv", err)
	}
}

func TestEndpointJoinPath(t *testing.T) {
	t.Parallel()

	client, err := New("https://example.com/licenses", "acme")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	endpoint, err := client.endpoint("/api/v1/validate")
	if err != nil {
		t.Fatalf("endpoint() error: %v", err)
	}
	if endpoint != "https://example.com/licenses/api/v1/validate" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestValidateValid(t *testing.T) {
	expires := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/validate" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}

		var req validateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Key != "TEST-KEY" || req.Product != "acme-widget" {
			t.Fatalf("request = %+v", req)
		}

		_ = json.NewEncoder(w).Encode(ValidationResult{
			Valid:     true,
			ExpiresAt: &expires,
			Product:   "acme-widget",
			Policy:    "trial",
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	result, err := client.Validate(context.Background(), "TEST-KEY")
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid license")
	}
	if result.Policy != "trial" {
		t.Fatalf("policy = %q", result.Policy)
	}
}

func TestValidateInvalidNotError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ValidationResult{
			Valid:  false,
			Reason: ReasonExpired,
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	result, err := client.Validate(context.Background(), "TEST-KEY")
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid license")
	}
	if result.Reason != ReasonExpired {
		t.Fatalf("reason = %q", result.Reason)
	}
}

func TestValidateRetryOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			writeAPIError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithRetry(2, 10*time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	result, err := client.Validate(context.Background(), "TEST-KEY")
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid after retry")
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRegistryCredentialsForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeAPIError(w, http.StatusForbidden, "expired")
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, err = client.RegistryCredentials(context.Background(), "TEST-KEY")
	var licErr *LicenseError
	if !errors.As(err, &licErr) {
		t.Fatalf("error = %T %v, want *LicenseError", err, err)
	}
	if licErr.Reason != ReasonExpired {
		t.Fatalf("reason = %q", licErr.Reason)
	}
}

func TestRegistryCredentialsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(registryCredentialsResponse{
			Registry:  "harbor.example.com",
			Username:  "robot$user",
			Secret:    "secret",
			ExpiresAt: 1725350400,
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	creds, err := client.RegistryCredentials(context.Background(), "TEST-KEY")
	if err != nil {
		t.Fatalf("RegistryCredentials() error: %v", err)
	}
	if creds.Registry != "harbor.example.com" {
		t.Fatalf("registry = %q", creds.Registry)
	}
}

func TestHealthReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case "/readyz":
			writeAPIError(w, http.StatusServiceUnavailable, "database unavailable")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health() error: %v", err)
	}

	if err := client.Ready(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Ready() error = %v, want ErrUnavailable", err)
	}
}

func TestAPIErrorIs(t *testing.T) {
	t.Parallel()

	err := &APIError{StatusCode: http.StatusTooManyRequests, Message: "rate limit exceeded"}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatal("expected ErrRateLimited")
	}
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func TestMain(m *testing.M) {
	// Ensure env vars from other tests don't leak.
	_ = os.Unsetenv(envURL)
	_ = os.Unsetenv(envProduct)
	os.Exit(m.Run())
}

func TestUserAgentHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("User-Agent"), "custom-agent") {
			t.Fatalf("user-agent = %q", r.Header.Get("User-Agent"))
		}
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
	}))
	defer server.Close()

	client, err := New(server.URL, "acme-widget", WithUserAgent("custom-agent/1.0"), WithRetry(1, time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if _, err := client.Validate(context.Background(), "TEST-KEY"); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
}
