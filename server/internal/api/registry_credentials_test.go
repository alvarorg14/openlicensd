package api_test

import (
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

func TestRegistryCredentialsRouteDisabled(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	passwordHash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:     databaseURL,
		SessionTTLHours: 24,
		CookieSecure:    false,
		LocalLoginEnabled: true,
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        fmt.Sprintf("admin-%d@example.com", time.Now().UnixNano()),
			Name:         "Test Admin",
			PasswordHash: passwordHash,
		},
		Harbor: config.HarborConfig{
			Enabled: false,
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

	srv := api.New(ctx, cfg, st)
	handler := srv.Router(nil)

	resp := doJSON(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": "invalid-key",
	}, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", resp.Code, resp.Body.String())
	}
}

func TestRegistryCredentialsEnabled(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	harborServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         99,
				"name":       "robot$myproject+openlicensd-test",
				"secret":     "issued-secret",
				"expires_at": time.Now().Add(24 * time.Hour).Unix(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots":
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(harborServer.Close)

	passwordHash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := fmt.Sprintf("admin-%d@example.com", time.Now().UnixNano())

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:     databaseURL,
		SessionTTLHours: 24,
		CookieSecure:    false,
		LocalLoginEnabled: true,
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        email,
			Name:         "Test Admin",
			PasswordHash: passwordHash,
		},
		Harbor: config.HarborConfig{
			Enabled:           true,
			URL:               harborServer.URL,
			AdminUsername:     "harbor-admin",
			AdminPassword:     "harbor-secret",
			Projects:          []string{"myproject"},
			RobotDurationDays: 1,
			RobotNamePrefix:   "openlicensd",
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

	if userCount > 0 {
		if _, err := st.CreateUser(ctx, email, cfg.BootstrapAdmin.Name, &passwordHash, store.RoleAdmin, store.AuthProviderLocal, nil); err != nil {
			t.Fatalf("create test admin: %v", err)
		}
	}

	srv := api.New(ctx, cfg, st)
	handler := srv.Router(nil)
	cookies := login(t, handler, email, "test-password")

	productCode := fmt.Sprintf("harbor-product-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Harbor Product",
		"code": productCode,
	}, cookies)
	if productResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", productResp.Code, productResp.Body.String())
	}

	var product map[string]any
	if err := json.Unmarshal(productResp.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product: %v", err)
	}

	policyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       product["id"],
		"name":             "Default",
		"expiration_basis": "on_creation",
	}, cookies)
	if policyResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", policyResp.Code, policyResp.Body.String())
	}

	var policy map[string]any
	if err := json.Unmarshal(policyResp.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy: %v", err)
	}

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "harbor-test",
		"product_id": product["id"],
		"policy_id":  policy["id"],
	}, cookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode license: %v", err)
	}

	rawKey := created["key"].(string)

	invalidResp := doJSON(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": "invalid-key",
	}, nil)
	if invalidResp.Code != http.StatusForbidden {
		t.Fatalf("invalid key status=%d want 403", invalidResp.Code)
	}

	validResp := doJSON(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": rawKey,
	}, nil)
	if validResp.Code != http.StatusOK {
		t.Fatalf("valid key status=%d body=%s", validResp.Code, validResp.Body.String())
	}
}
