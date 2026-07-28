package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/api"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/license"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

func TestAPIIntegration(t *testing.T) {
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

	createBody := map[string]any{
		"label": "integration-test",
	}
	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", createBody, token)
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

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses", nil, token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list licenses status=%d", listResp.Code)
	}

	validateResp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": rawKey}, "")
	if validateResp.Code != http.StatusOK {
		t.Fatalf("validate status=%d", validateResp.Code)
	}

	var validation license.ValidationResult
	if err := json.Unmarshal(validateResp.Body.Bytes(), &validation); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("expected valid license, got %+v", validation)
	}

	expiresAt := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	expiredResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "expired",
		"expires_at": expiresAt,
	}, token)
	if expiredResp.Code != http.StatusCreated {
		t.Fatalf("create expired license status=%d", expiredResp.Code)
	}

	var expiredCreated map[string]any
	if err := json.Unmarshal(expiredResp.Body.Bytes(), &expiredCreated); err != nil {
		t.Fatalf("decode expired create response: %v", err)
	}

	expiredKey := expiredCreated["key"].(string)
	expiredValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": expiredKey}, "")
	var expiredValidation license.ValidationResult
	if err := json.Unmarshal(expiredValidate.Body.Bytes(), &expiredValidation); err != nil {
		t.Fatalf("decode expired validate response: %v", err)
	}
	if expiredValidation.Valid || expiredValidation.Reason != "expired" {
		t.Fatalf("expected expired license, got %+v", expiredValidation)
	}
}

func login(t *testing.T, handler http.Handler, username, password string) string {
	t.Helper()

	resp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, "")

	if resp.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	return body.Token
}

func doJSON(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
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
