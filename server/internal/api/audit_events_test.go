package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/store"
)

func TestAuditEventOnProductCreateSession(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("audit-product-%d", time.Now().UnixNano())
	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Audit Product",
		"code": productCode,
	}, adminCookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var product map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product: %v", err)
	}

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/audit-events?action=product.create&search=Audit+Product", nil, adminCookies)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list audit events status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	var page map[string]any
	if err := json.Unmarshal(listResp.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode audit list: %v", err)
	}
	items, ok := page["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected audit events for product create, got %#v", page["items"])
	}

	event := items[0].(map[string]any)
	if event["action"] != "product.create" {
		t.Fatalf("action=%v want product.create", event["action"])
	}
	if event["resource_type"] != "product" {
		t.Fatalf("resource_type=%v want product", event["resource_type"])
	}
	if event["resource_id"] != product["id"] {
		t.Fatalf("resource_id=%v want %v", event["resource_id"], product["id"])
	}
	if event["auth_method"] != "session" {
		t.Fatalf("auth_method=%v want session", event["auth_method"])
	}
	if event["actor_user_id"] == nil {
		t.Fatal("expected actor_user_id for session auth")
	}
	if event["actor_email"] == nil {
		t.Fatal("expected actor_email for session auth")
	}
}

func TestAuditEventOnProductCreateBearerToken(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler

	operatorToken := createTestAPIToken(t, env.Store, "audit-op-"+fmt.Sprint(time.Now().UnixNano()), store.RoleOperator)

	productCode := fmt.Sprintf("audit-bearer-%d", time.Now().UnixNano())
	createResp := doJSONWithToken(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Bearer Product",
		"code": productCode,
	}, operatorToken)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	adminToken := createTestAPIToken(t, env.Store, "audit-admin-"+fmt.Sprint(time.Now().UnixNano()), store.RoleAdmin)
	listResp := doJSONWithToken(t, handler, http.MethodGet, "/api/v1/audit-events?action=product.create&search=Bear", nil, adminToken)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list audit events status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	var page map[string]any
	if err := json.Unmarshal(listResp.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode audit list: %v", err)
	}
	items := page["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected audit event for bearer product create")
	}

	event := items[0].(map[string]any)
	if event["auth_method"] != "api_token" {
		t.Fatalf("auth_method=%v want api_token", event["auth_method"])
	}
	if event["actor_user_id"] != nil {
		t.Fatalf("actor_user_id=%v want null for api token", event["actor_user_id"])
	}
	if event["actor_token_id"] == nil {
		t.Fatal("expected actor_token_id for api token auth")
	}
}

func TestAuditEventListRequiresAdmin(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	viewerEmail := fmt.Sprintf("audit-viewer-%d", time.Now().UnixNano())
	createViewer := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    viewerEmail,
		"name":     "Viewer",
		"password": "viewer-password",
		"role":     "viewer",
	}, adminCookies)
	if createViewer.Code != http.StatusCreated {
		t.Fatalf("create viewer status=%d", createViewer.Code)
	}
	viewerCookies := login(t, handler, viewerEmail, "viewer-password")

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/audit-events", nil, viewerCookies)
	if listResp.Code != http.StatusForbidden {
		t.Fatalf("viewer list audit events status=%d want 403", listResp.Code)
	}
}

func TestFailedMutationDoesNotCreateAuditEvent(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	beforeResp := doJSON(t, handler, http.MethodGet, "/api/v1/audit-events", nil, adminCookies)
	if beforeResp.Code != http.StatusOK {
		t.Fatalf("list audit events status=%d", beforeResp.Code)
	}
	var before map[string]any
	if err := json.Unmarshal(beforeResp.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode before: %v", err)
	}
	beforeTotal := before["total"].(float64)

	failResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "",
		"code": "",
	}, adminCookies)
	if failResp.Code != http.StatusBadRequest {
		t.Fatalf("create product status=%d want 400", failResp.Code)
	}

	afterResp := doJSON(t, handler, http.MethodGet, "/api/v1/audit-events", nil, adminCookies)
	if afterResp.Code != http.StatusOK {
		t.Fatalf("list audit events status=%d", afterResp.Code)
	}
	var after map[string]any
	if err := json.Unmarshal(afterResp.Body.Bytes(), &after); err != nil {
		t.Fatalf("decode after: %v", err)
	}
	afterTotal := after["total"].(float64)
	if afterTotal != beforeTotal {
		t.Fatalf("audit total changed from %v to %v after failed mutation", beforeTotal, afterTotal)
	}
}
