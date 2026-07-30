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

	"github.com/openlicensd/openlicensd/server/internal/api"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/store"
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
		Addr:            ":8080",
		DatabaseURL:     databaseURL,
		SessionTTLHours: 24,
		CookieSecure:    false,
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

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	hash := cfg.BootstrapAdmin.PasswordHash
	if _, err := st.CreateUser(ctx, email, cfg.BootstrapAdmin.Name, &hash, store.RoleAdmin, store.AuthProviderLocal, nil); err != nil {
		t.Fatalf("create test admin: %v", err)
	}

	srv := api.New(cfg, st)

	return testEnv{
		Handler:  srv.Router(nil),
		Store:    st,
		Email:    email,
		Password: "test-password",
	}
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
