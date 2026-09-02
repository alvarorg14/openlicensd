package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAuditRoutesCoverage(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler

	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatal("handler does not implement chi.Routes")
	}

	var missing []string
	err := chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !isMutatingMethodForTest(method) {
			return nil
		}
		if !strings.HasPrefix(route, "/api/v1/") {
			return nil
		}
		if isExcludedFromAudit(route) {
			return nil
		}

		key := method + " " + route
		if !auditRouteRegistered(key) {
			missing = append(missing, key)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	if len(missing) > 0 {
		t.Fatalf("mutating routes missing from auditRoutes:\n%s", strings.Join(missing, "\n"))
	}
}

func isMutatingMethodForTest(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isExcludedFromAudit(route string) bool {
	switch route {
	case "/api/v1/auth/login",
		"/api/v1/auth/oidc/login",
		"/api/v1/auth/oidc/callback",
		"/api/v1/validate",
		"/api/v1/registry-credentials":
		return true
	default:
		return false
	}
}

// auditRouteRegistered mirrors api.auditRoutes keys; keep in sync with audit.go.
func auditRouteRegistered(key string) bool {
	registered := map[string]bool{
		"POST /api/v1/auth/logout":                          true,
		"POST /api/v1/auth/password":                        true,
		"POST /api/v1/licenses":                             true,
		"PATCH /api/v1/licenses/{id}":                       true,
		"DELETE /api/v1/licenses/{id}":                      true,
		"PATCH /api/v1/licenses/{id}/revoke":                true,
		"PATCH /api/v1/licenses/{id}/unrevoke":              true,
		"PATCH /api/v1/licenses/{id}/machines/{machineId}":  true,
		"DELETE /api/v1/licenses/{id}/machines/{machineId}": true,
		"POST /api/v1/products":                             true,
		"PATCH /api/v1/products/{id}":                       true,
		"DELETE /api/v1/products/{id}":                    true,
		"POST /api/v1/policies":                             true,
		"PATCH /api/v1/policies/{id}":                       true,
		"DELETE /api/v1/policies/{id}":                      true,
		"POST /api/v1/users":                                true,
		"PATCH /api/v1/users/{id}":                          true,
		"PATCH /api/v1/users/{id}/password":                 true,
		"PATCH /api/v1/users/{id}/disable":                  true,
		"PATCH /api/v1/users/{id}/enable":                   true,
		"DELETE /api/v1/users/{id}":                         true,
		"POST /api/v1/api-tokens":                           true,
		"PATCH /api/v1/api-tokens/{id}/revoke":              true,
		"DELETE /api/v1/api-tokens/{id}":                    true,
	}
	return registered[key]
}
