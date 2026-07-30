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
		DatabaseURL:       databaseURL,
		AdminUser:         "admin",
		AdminPasswordHash: passwordHash,
		JWTSecret:         "test-jwt-secret",
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

	srv := api.New(cfg, st)
	handler := srv.Router(nil)

	resp := doJSON(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": "invalid-key",
	}, "")
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

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:       databaseURL,
		AdminUser:         "admin",
		AdminPasswordHash: passwordHash,
		JWTSecret:         "test-jwt-secret",
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

	srv := api.New(cfg, st)
	handler := srv.Router(nil)
	token := login(t, handler, "admin", "test-password")

	productCode := fmt.Sprintf("registry-product-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Registry Product",
		"code": productCode,
	}, token)
	if productResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", productResp.Code, productResp.Body.String())
	}

	var product map[string]any
	if err := json.Unmarshal(productResp.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	policyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id": product["id"],
		"name":       "Perpetual",
	}, token)
	if policyResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", policyResp.Code, policyResp.Body.String())
	}

	var policy map[string]any
	if err := json.Unmarshal(policyResp.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy response: %v", err)
	}

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "registry-credentials-test",
		"product_id": product["id"],
		"policy_id":  policy["id"],
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	rawKey, ok := created["key"].(string)
	if !ok || rawKey == "" {
		t.Fatalf("expected raw key in create response")
	}

	invalidResp := doJSON(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": "not-a-real-key",
	}, "")
	if invalidResp.Code != http.StatusForbidden {
		t.Fatalf("invalid key status=%d want 403 body=%s", invalidResp.Code, invalidResp.Body.String())
	}

	var invalidBody map[string]string
	if err := json.Unmarshal(invalidResp.Body.Bytes(), &invalidBody); err != nil {
		t.Fatalf("decode invalid response: %v", err)
	}
	if invalidBody["error"] != "not_found" {
		t.Fatalf("error=%q want not_found", invalidBody["error"])
	}

	validResp := doJSON(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": rawKey,
	}, "")
	if validResp.Code != http.StatusOK {
		t.Fatalf("valid key status=%d want 200 body=%s", validResp.Code, validResp.Body.String())
	}

	var creds struct {
		Registry  string `json:"registry"`
		Username  string `json:"username"`
		Secret    string `json:"secret"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(validResp.Body.Bytes(), &creds); err != nil {
		t.Fatalf("decode credentials response: %v", err)
	}

	if creds.Registry == "" {
		t.Fatalf("expected registry host")
	}
	if creds.Username != "robot$myproject+openlicensd-test" {
		t.Fatalf("username=%q", creds.Username)
	}
	if creds.Secret != "issued-secret" {
		t.Fatalf("secret=%q", creds.Secret)
	}
	if creds.ExpiresAt == 0 {
		t.Fatalf("expected expires_at")
	}
}
