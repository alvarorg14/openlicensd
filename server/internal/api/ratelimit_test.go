package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/api"
	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/store"
)

func setupRateLimitTestEnv(t *testing.T, rateLimit config.RateLimitConfig) http.Handler {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:       databaseURL,
		SessionTTLHours:   24,
		CookieSecure:      false,
		LocalLoginEnabled: true,
		RateLimit:         rateLimit,
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)

	srv, err := api.New(ctx, cfg, st, testLogger())
	if err != nil {
		t.Fatalf("api server: %v", err)
	}
	return srv.Router(nil)
}

func doJSONWithRemoteAddr(t *testing.T, handler http.Handler, method, path string, body any, cookies []*http.Cookie, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	if isUnsafeMethod(method) {
		for _, cookie := range cookies {
			if cookie.Name == auth.CSRFCookieName {
				req.Header.Set(auth.CSRFHeaderName, cookie.Value)
				break
			}
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRateLimitValidateReturns429(t *testing.T) {
	handler := setupRateLimitTestEnv(t, config.RateLimitConfig{
		Enabled:         true,
		PublicPerMinute: 60,
		PublicBurst:     2,
		LoginPerMinute:  30,
		LoginBurst:      10,
		IdleMinutes:     10,
	})

	remoteAddr := fmt.Sprintf("203.0.113.%d:12345", time.Now().UnixNano()%200+1)

	for i := 0; i < 2; i++ {
		resp := doJSONWithRemoteAddr(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
			"key": "invalid-key",
		}, nil, remoteAddr)
		if resp.Code != http.StatusOK {
			t.Fatalf("request %d status=%d body=%s", i, resp.Code, resp.Body.String())
		}
	}

	resp := doJSONWithRemoteAddr(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key": "invalid-key",
	}, nil, remoteAddr)
	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d want 429 body=%s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "rate limit exceeded" {
		t.Fatalf("error=%q", body["error"])
	}
}

func TestRateLimitDisabledNeverReturns429(t *testing.T) {
	handler := setupRateLimitTestEnv(t, config.RateLimitConfig{
		Enabled: false,
	})

	remoteAddr := fmt.Sprintf("203.0.113.%d:12345", time.Now().UnixNano()%200+1)

	for i := 0; i < 5; i++ {
		resp := doJSONWithRemoteAddr(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
			"key": "invalid-key",
		}, nil, remoteAddr)
		if resp.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d unexpectedly rate limited", i)
		}
	}
}

func TestRateLimitLoginReturns429(t *testing.T) {
	env := setupTestEnv(t)
	handler := setupRateLimitTestEnv(t, config.RateLimitConfig{
		Enabled:         true,
		PublicPerMinute: 600,
		PublicBurst:     60,
		LoginPerMinute:  30,
		LoginBurst:      2,
		IdleMinutes:     10,
	})

	remoteAddr := fmt.Sprintf("198.51.100.%d:12345", time.Now().UnixNano()%200+1)

	for i := 0; i < 2; i++ {
		resp := doJSONWithRemoteAddr(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"email":    env.Email,
			"password": "wrong-password",
		}, nil, remoteAddr)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("request %d status=%d body=%s", i, resp.Code, resp.Body.String())
		}
	}

	resp := doJSONWithRemoteAddr(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    env.Email,
		"password": "wrong-password",
	}, nil, remoteAddr)
	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d want 429 body=%s", resp.Code, resp.Body.String())
	}
}
