package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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

type testEnv struct {
	Handler  http.Handler
	Store    *store.Store
	Email    string
	Password string
}

func setupTestEnv(t *testing.T) testEnv {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	passwordHash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := fmt.Sprintf("admin-%d@example.com", time.Now().UnixNano())

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:       databaseURL,
		SessionTTLHours:   24,
		CookieSecure:      false,
		LocalLoginEnabled: true,
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        email,
			Name:         "Test Admin",
			PasswordHash: passwordHash,
		},
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)

	userCount, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Bootstrap only seeds when the users table is empty; otherwise create a dedicated test admin.
	if userCount > 0 {
		hash := cfg.BootstrapAdmin.PasswordHash
		if _, err := st.CreateUser(ctx, email, cfg.BootstrapAdmin.Name, &hash, store.RoleAdmin, store.AuthProviderLocal, nil); err != nil {
			t.Fatalf("create test admin: %v", err)
		}
	}

	srv, err := api.New(ctx, cfg, st, testLogger())
	if err != nil {
		t.Fatalf("api server: %v", err)
	}

	return testEnv{
		Handler:  srv.Router(nil),
		Store:    st,
		Email:    email,
		Password: "test-password",
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func login(t *testing.T, handler http.Handler, email, password string) []*http.Cookie {
	t.Helper()

	resp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, nil)

	if resp.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", resp.Code, resp.Body.String())
	}

	return resp.Result().Cookies()
}

func doJSON(t *testing.T, handler http.Handler, method, path string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
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

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func createTestAPIToken(t *testing.T, st *store.Store, name string, role store.Role) string {
	t.Helper()

	raw, hash, prefix, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate api token: %v", err)
	}

	ctx := context.Background()
	if _, err := st.CreateAPIToken(ctx, name, hash, prefix, role, nil, nil); err != nil {
		t.Fatalf("create api token: %v", err)
	}
	return raw
}

func doJSONWithToken(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
