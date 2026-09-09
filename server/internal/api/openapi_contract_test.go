package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/api"
	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/pb33f/libopenapi"
	oapivalidator "github.com/pb33f/libopenapi-validator"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type openAPIContractSuite struct {
	t         *testing.T
	validator oapivalidator.Validator
	handler   http.Handler
	cookies   []*http.Cookie
	exercised map[string]bool
}

func openAPISpecPath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "docs", "openapi.yaml"))
}

func loadOpenAPIValidator(t *testing.T) oapivalidator.Validator {
	t.Helper()

	spec, err := os.ReadFile(openAPISpecPath(t))
	if err != nil {
		t.Fatalf("read openapi spec: %v", err)
	}

	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatalf("parse openapi spec: %v", err)
	}

	validator, validatorErrs := oapivalidator.NewValidator(document)
	if len(validatorErrs) > 0 {
		t.Fatalf("create openapi validator: %v", validatorErrs)
	}
	return validator
}

func collectOperationIDs(t *testing.T, specPath string) []string {
	t.Helper()

	spec, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read openapi spec: %v", err)
	}

	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatalf("parse openapi spec: %v", err)
	}

	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatalf("build openapi model: %v", err)
	}

	var ids []string
	for _, pathItem := range model.Model.Paths.PathItems.FromOldest() {
		for _, op := range []*v3.Operation{
			pathItem.Get,
			pathItem.Put,
			pathItem.Post,
			pathItem.Delete,
			pathItem.Options,
			pathItem.Head,
			pathItem.Patch,
			pathItem.Trace,
		} {
			if op != nil && op.OperationId != "" {
				ids = append(ids, op.OperationId)
			}
		}
	}
	sort.Strings(ids)
	return ids
}

func newOpenAPIContractSuite(t *testing.T, handler http.Handler, validator oapivalidator.Validator) *openAPIContractSuite {
	return &openAPIContractSuite{
		t:         t,
		validator: validator,
		handler:   handler,
		exercised: make(map[string]bool),
	}
}

func (s *openAPIContractSuite) check(operationID string, method, path string, body any, cookies []*http.Cookie, bearerToken string) contractExchange {
	s.t.Helper()
	s.exercised[operationID] = true

	ex := performHTTP(s.t, s.handler, method, path, body, cookies, bearerToken)
	assertOpenAPIResponse(s.t, s.validator, ex)
	return ex
}

func (s *openAPIContractSuite) assertExercised(specPath string) {
	s.t.Helper()

	for _, id := range collectOperationIDs(s.t, specPath) {
		if !s.exercised[id] {
			s.t.Errorf("operation %q has no contract scenario", id)
		}
	}
}

func decodeJSONMap(t *testing.T, ex contractExchange) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(ex.Recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode json response: %v body=%s", err, ex.Recorder.Body.String())
	}
	return out
}

func requireStatus(t *testing.T, ex contractExchange, want int) {
	t.Helper()
	if ex.Recorder.Code != want {
		t.Fatalf("%s %s status=%d want %d body=%s", ex.Request.Method, ex.Request.URL.Path, ex.Recorder.Code, want, ex.Recorder.Body.String())
	}
}

func TestOpenAPIContract(t *testing.T) {
	specPath := openAPISpecPath(t)
	validator := loadOpenAPIValidator(t)
	env := setupTestEnv(t)

	suite := newOpenAPIContractSuite(t, env.Handler, validator)

	// Health probes
	requireStatus(t, suite.check("healthz", http.MethodGet, "/healthz", nil, nil, ""), http.StatusOK)
	requireStatus(t, suite.check("readyz", http.MethodGet, "/readyz", nil, nil, ""), http.StatusOK)

	// Auth providers and disabled OIDC routes
	requireStatus(t, suite.check("listAuthProviders", http.MethodGet, "/api/v1/auth/providers", nil, nil, ""), http.StatusOK)
	requireStatus(t, suite.check("oidcLogin", http.MethodGet, "/api/v1/auth/oidc/login", nil, nil, ""), http.StatusNotFound)
	requireStatus(t, suite.check("oidcCallback", http.MethodGet, "/api/v1/auth/oidc/callback", nil, nil, ""), http.StatusNotFound)

	loginEx := suite.check("login", http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    env.Email,
		"password": env.Password,
	}, nil, "")
	requireStatus(t, loginEx, http.StatusOK)
	suite.cookies = loginEx.Recorder.Result().Cookies()

	requireStatus(t, suite.check("getCurrentUser", http.MethodGet, "/api/v1/auth/me", nil, suite.cookies, ""), http.StatusOK)

	productCode := fmt.Sprintf("contract-product-%d", time.Now().UnixNano())
	productEx := suite.check("createProduct", http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Contract Product",
		"code": productCode,
	}, suite.cookies, "")
	requireStatus(t, productEx, http.StatusCreated)
	product := decodeJSONMap(t, productEx)
	productID := product["id"].(string)

	requireStatus(t, suite.check("listProducts", http.MethodGet, "/api/v1/products", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("getProduct", http.MethodGet, "/api/v1/products/"+productID, nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("updateProduct", http.MethodPatch, "/api/v1/products/"+productID, map[string]any{
		"name": "Contract Product Updated",
	}, suite.cookies, ""), http.StatusOK)

	policyEx := suite.check("createPolicy", http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       productID,
		"name":             "Contract Policy",
		"expiration_basis": "on_creation",
	}, suite.cookies, "")
	requireStatus(t, policyEx, http.StatusCreated)
	policy := decodeJSONMap(t, policyEx)
	policyID := policy["id"].(string)

	fpPolicyEx := suite.check("createPolicy", http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       productID,
		"name":             "One Seat",
		"expiration_basis": "on_creation",
		"max_activations":  1,
	}, suite.cookies, "")
	requireStatus(t, fpPolicyEx, http.StatusCreated)
	fpPolicy := decodeJSONMap(t, fpPolicyEx)
	fpPolicyID := fpPolicy["id"].(string)

	requireStatus(t, suite.check("listPolicies", http.MethodGet, "/api/v1/policies", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("getPolicy", http.MethodGet, "/api/v1/policies/"+policyID, nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("updatePolicy", http.MethodPatch, "/api/v1/policies/"+policyID, map[string]any{
		"name": "Contract Policy Updated",
	}, suite.cookies, ""), http.StatusOK)

	requireStatus(t, suite.check("getLicenseStats", http.MethodGet, "/api/v1/licenses/stats", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("listLicenses", http.MethodGet, "/api/v1/licenses", nil, suite.cookies, ""), http.StatusOK)

	licenseEx := suite.check("createLicense", http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "contract-license",
		"product_id": productID,
		"policy_id":  policyID,
	}, suite.cookies, "")
	requireStatus(t, licenseEx, http.StatusCreated)
	license := decodeJSONMap(t, licenseEx)
	licenseID := license["id"].(string)
	rawKey := license["key"].(string)

	fpLicenseEx := suite.check("createLicense", http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "contract-fp-license",
		"product_id": productID,
		"policy_id":  fpPolicyID,
	}, suite.cookies, "")
	requireStatus(t, fpLicenseEx, http.StatusCreated)
	fpLicense := decodeJSONMap(t, fpLicenseEx)
	fpLicenseID := fpLicense["id"].(string)
	fpKey := fpLicense["key"].(string)

	requireStatus(t, suite.check("getLicense", http.MethodGet, "/api/v1/licenses/"+licenseID, nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("updateLicense", http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{
		"label": "contract-license-updated",
	}, suite.cookies, ""), http.StatusOK)

	requireStatus(t, suite.check("validateLicense", http.MethodPost, "/api/v1/validate", map[string]string{
		"key":     rawKey,
		"product": productCode,
	}, nil, ""), http.StatusOK)

	notFoundEx := performHTTP(t, env.Handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key": "invalid-key",
	}, nil, "")
	requireStatus(t, notFoundEx, http.StatusOK)
	assertOpenAPIResponse(t, validator, notFoundEx)

	fpRequiredEx := performHTTP(t, env.Handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key":     fpKey,
		"product": productCode,
	}, nil, "")
	requireStatus(t, fpRequiredEx, http.StatusOK)
	assertOpenAPIResponse(t, validator, fpRequiredEx)

	requireStatus(t, suite.check("revokeLicense", http.MethodPatch, "/api/v1/licenses/"+licenseID+"/revoke", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("unrevokeLicense", http.MethodPatch, "/api/v1/licenses/"+licenseID+"/unrevoke", nil, suite.cookies, ""), http.StatusOK)

	requireStatus(t, suite.check("validateLicense", http.MethodPost, "/api/v1/validate", map[string]string{
		"key":         fpKey,
		"product":     productCode,
		"fingerprint": "contract-machine",
		"hostname":    "contract.local",
	}, nil, ""), http.StatusOK)

	machinesEx := suite.check("listLicenseMachines", http.MethodGet, "/api/v1/licenses/"+fpLicenseID+"/machines", nil, suite.cookies, "")
	requireStatus(t, machinesEx, http.StatusOK)
	machinesPage := decodeJSONMap(t, machinesEx)
	items, ok := machinesPage["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected at least one machine, got %#v", machinesPage)
	}
	machineID := items[0].(map[string]any)["id"].(string)

	requireStatus(t, suite.check("updateLicenseMachine", http.MethodPatch, "/api/v1/licenses/"+fpLicenseID+"/machines/"+machineID, map[string]any{
		"name": "renamed-contract-machine",
	}, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("releaseLicenseMachine", http.MethodDelete, "/api/v1/licenses/"+fpLicenseID+"/machines/"+machineID, nil, suite.cookies, ""), http.StatusOK)

	viewerEmail := fmt.Sprintf("contract-viewer-%d@example.com", time.Now().UnixNano())
	viewerEx := suite.check("createUser", http.MethodPost, "/api/v1/users", map[string]any{
		"email":    viewerEmail,
		"name":     "Contract Viewer",
		"password": "viewer-pass",
		"role":     "viewer",
	}, suite.cookies, "")
	requireStatus(t, viewerEx, http.StatusCreated)
	viewer := decodeJSONMap(t, viewerEx)
	viewerID := viewer["id"].(string)

	deleteEmail := fmt.Sprintf("contract-delete-%d@example.com", time.Now().UnixNano())
	deleteUserEx := suite.check("createUser", http.MethodPost, "/api/v1/users", map[string]any{
		"email":    deleteEmail,
		"name":     "Contract Delete",
		"password": "delete-pass",
		"role":     "viewer",
	}, suite.cookies, "")
	requireStatus(t, deleteUserEx, http.StatusCreated)
	deleteUser := decodeJSONMap(t, deleteUserEx)
	deleteUserID := deleteUser["id"].(string)

	requireStatus(t, suite.check("listUsers", http.MethodGet, "/api/v1/users", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("getUser", http.MethodGet, "/api/v1/users/"+viewerID, nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("updateUser", http.MethodPatch, "/api/v1/users/"+viewerID, map[string]any{
		"email": viewerEmail,
		"name":  "Contract Viewer Updated",
		"role":  "viewer",
	}, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("setUserPassword", http.MethodPatch, "/api/v1/users/"+viewerID+"/password", map[string]string{
		"password": "new-viewer-pass",
	}, suite.cookies, ""), http.StatusNoContent)
	requireStatus(t, suite.check("disableUser", http.MethodPatch, "/api/v1/users/"+viewerID+"/disable", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("enableUser", http.MethodPatch, "/api/v1/users/"+viewerID+"/enable", nil, suite.cookies, ""), http.StatusOK)

	tokenSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tokenMeEx := suite.check("createAPIToken", http.MethodPost, "/api/v1/api-tokens", map[string]any{
		"name": "contract-me-" + tokenSuffix,
		"role": "viewer",
	}, suite.cookies, "")
	requireStatus(t, tokenMeEx, http.StatusCreated)
	tokenMe := decodeJSONMap(t, tokenMeEx)
	rawTokenMe := tokenMe["token"].(string)

	requireStatus(t, suite.check("getCurrentUser", http.MethodGet, "/api/v1/auth/me", nil, nil, rawTokenMe), http.StatusOK)

	requireStatus(t, suite.check("listAPITokens", http.MethodGet, "/api/v1/api-tokens", nil, suite.cookies, ""), http.StatusOK)

	tokenRevokeEx := suite.check("createAPIToken", http.MethodPost, "/api/v1/api-tokens", map[string]any{
		"name": "contract-revoke-" + tokenSuffix,
		"role": "viewer",
	}, suite.cookies, "")
	requireStatus(t, tokenRevokeEx, http.StatusCreated)
	tokenRevoke := decodeJSONMap(t, tokenRevokeEx)
	tokenRevokeID := tokenRevoke["id"].(string)

	requireStatus(t, suite.check("revokeAPIToken", http.MethodPatch, "/api/v1/api-tokens/"+tokenRevokeID+"/revoke", nil, suite.cookies, ""), http.StatusOK)
	requireStatus(t, suite.check("deleteAPIToken", http.MethodDelete, "/api/v1/api-tokens/"+tokenRevokeID, nil, suite.cookies, ""), http.StatusNoContent)

	requireStatus(t, suite.check("listAuditEvents", http.MethodGet, "/api/v1/audit-events", nil, suite.cookies, ""), http.StatusOK)

	exerciseRegistryCredentialsContract(t, suite)

	requireStatus(t, suite.check("changeOwnPassword", http.MethodPost, "/api/v1/auth/password", map[string]string{
		"current_password": env.Password,
		"password":         "contract-admin-pass",
	}, suite.cookies, ""), http.StatusNoContent)

	requireStatus(t, suite.check("deleteLicense", http.MethodDelete, "/api/v1/licenses/"+licenseID, nil, suite.cookies, ""), http.StatusNoContent)
	requireStatus(t, suite.check("deleteLicense", http.MethodDelete, "/api/v1/licenses/"+fpLicenseID, nil, suite.cookies, ""), http.StatusNoContent)
	requireStatus(t, suite.check("deletePolicy", http.MethodDelete, "/api/v1/policies/"+policyID, nil, suite.cookies, ""), http.StatusNoContent)
	requireStatus(t, suite.check("deletePolicy", http.MethodDelete, "/api/v1/policies/"+fpPolicyID, nil, suite.cookies, ""), http.StatusNoContent)
	requireStatus(t, suite.check("deleteProduct", http.MethodDelete, "/api/v1/products/"+productID, nil, suite.cookies, ""), http.StatusNoContent)
	requireStatus(t, suite.check("deleteUser", http.MethodDelete, "/api/v1/users/"+deleteUserID, nil, suite.cookies, ""), http.StatusNoContent)

	requireStatus(t, suite.check("logout", http.MethodPost, "/api/v1/auth/logout", nil, suite.cookies, ""), http.StatusNoContent)

	suite.assertExercised(specPath)
}

func exerciseRegistryCredentialsContract(t *testing.T, suite *openAPIContractSuite) {
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

	email := fmt.Sprintf("harbor-contract-%d@example.com", time.Now().UnixNano())
	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:       databaseURL,
		SessionTTLHours:   24,
		CookieSecure:      false,
		LocalLoginEnabled: true,
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        email,
			Name:         "Harbor Contract Admin",
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

	srv, err := api.New(ctx, cfg, st, testLogger())
	if err != nil {
		t.Fatalf("api server: %v", err)
	}
	handler := srv.Router(nil)
	cookies := login(t, handler, email, "test-password")

	productCode := fmt.Sprintf("harbor-contract-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Harbor Contract Product",
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

	licenseResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "harbor-contract",
		"product_id": product["id"],
		"policy_id":  policy["id"],
	}, cookies)
	if licenseResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", licenseResp.Code, licenseResp.Body.String())
	}
	var license map[string]any
	if err := json.Unmarshal(licenseResp.Body.Bytes(), &license); err != nil {
		t.Fatalf("decode license: %v", err)
	}

	invalidEx := performHTTP(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": "invalid-key",
	}, nil, "")
	requireStatus(t, invalidEx, http.StatusForbidden)
	assertOpenAPIResponse(t, suite.validator, invalidEx)

	ex := performHTTP(t, handler, http.MethodPost, "/api/v1/registry-credentials", map[string]string{
		"key": license["key"].(string),
	}, nil, "")
	requireStatus(t, ex, http.StatusOK)
	suite.exercised["registryCredentials"] = true
	assertOpenAPIResponse(t, suite.validator, ex)
}
