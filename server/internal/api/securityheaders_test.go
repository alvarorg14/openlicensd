package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name         string
		cookieSecure bool
		wantHSTS     bool
	}{
		{
			name:         "cookie secure false omits HSTS",
			cookieSecure: false,
			wantHSTS:     false,
		},
		{
			name:         "cookie secure true sets HSTS",
			cookieSecure: true,
			wantHSTS:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nextCalled bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			})

			handler := securityHeaders(tt.cookieSecure)(next)

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if !nextCalled {
				t.Fatal("next handler was not called")
			}

			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
			}
			if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
				t.Fatalf("X-Frame-Options = %q, want %q", got, "DENY")
			}
			if got := rec.Header().Get("Content-Security-Policy"); got != contentSecurityPolicy {
				t.Fatalf("Content-Security-Policy = %q, want %q", got, contentSecurityPolicy)
			}

			hsts := rec.Header().Get("Strict-Transport-Security")
			if tt.wantHSTS {
				if hsts != "max-age=31536000" {
					t.Fatalf("Strict-Transport-Security = %q, want %q", hsts, "max-age=31536000")
				}
			} else if hsts != "" {
				t.Fatalf("Strict-Transport-Security = %q, want empty", hsts)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want %q", got, "application/json")
			}
		})
	}
}
