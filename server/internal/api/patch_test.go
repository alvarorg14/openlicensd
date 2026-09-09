package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestPatchLicensePartialUpdate(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productID, policyID := createPatchTestProductAndPolicy(t, handler, cookies)
	expiresAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	maxActivations := 3

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":            "patch-license",
		"product_id":       productID,
		"policy_id":        policyID,
		"expires_at":       expiresAt.Format(time.RFC3339),
		"max_activations":  maxActivations,
	}, cookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	licenseID := created["id"].(string)

	updateResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{
		"label": "updated-label",
	}, cookies)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("patch license status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}

	var updated map[string]any
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode patch response: %v", err)
	}
	if updated["label"] != "updated-label" {
		t.Fatalf("expected updated label, got %+v", updated["label"])
	}
	if updated["expires_at"] == nil {
		t.Fatal("expected expires_at to be preserved")
	}
	if updated["max_activations"] == nil {
		t.Fatal("expected max_activations to be preserved")
	}
	if int(updated["max_activations"].(float64)) != maxActivations {
		t.Fatalf("expected max_activations=%d, got %+v", maxActivations, updated["max_activations"])
	}

	clearExpiryResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{
		"expires_at": nil,
	}, cookies)
	if clearExpiryResp.Code != http.StatusOK {
		t.Fatalf("clear expires_at status=%d body=%s", clearExpiryResp.Code, clearExpiryResp.Body.String())
	}
	var clearedExpiry map[string]any
	if err := json.Unmarshal(clearExpiryResp.Body.Bytes(), &clearedExpiry); err != nil {
		t.Fatalf("decode clear expiry response: %v", err)
	}
	if clearedExpiry["expires_at"] != nil {
		t.Fatalf("expected expires_at to be cleared, got %+v", clearedExpiry["expires_at"])
	}

	clearMaxResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{
		"max_activations": nil,
	}, cookies)
	if clearMaxResp.Code != http.StatusOK {
		t.Fatalf("clear max_activations status=%d body=%s", clearMaxResp.Code, clearMaxResp.Body.String())
	}
	var clearedMax map[string]any
	if err := json.Unmarshal(clearMaxResp.Body.Bytes(), &clearedMax); err != nil {
		t.Fatalf("decode clear max response: %v", err)
	}
	if clearedMax["max_activations"] != nil {
		t.Fatalf("expected max_activations to be cleared, got %+v", clearedMax["max_activations"])
	}

	emptyResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{}, cookies)
	if emptyResp.Code != http.StatusBadRequest {
		t.Fatalf("empty patch status=%d body=%s", emptyResp.Code, emptyResp.Body.String())
	}

	emptyLabelResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{
		"label": "",
	}, cookies)
	if emptyLabelResp.Code != http.StatusBadRequest {
		t.Fatalf("empty label patch status=%d body=%s", emptyLabelResp.Code, emptyLabelResp.Body.String())
	}
}

func TestPatchProductPartialUpdate(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	description := "original description"
	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name":        "Patch Product",
		"code":        fmt.Sprintf("patch-product-%d", time.Now().UnixNano()),
		"description": description,
	}, cookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	productID := created["id"].(string)
	originalCode := created["code"].(string)

	nameOnlyResp := doJSON(t, handler, http.MethodPatch, "/api/v1/products/"+productID, map[string]any{
		"name": "Renamed Product",
	}, cookies)
	if nameOnlyResp.Code != http.StatusOK {
		t.Fatalf("patch product name status=%d body=%s", nameOnlyResp.Code, nameOnlyResp.Body.String())
	}
	var renamed map[string]any
	if err := json.Unmarshal(nameOnlyResp.Body.Bytes(), &renamed); err != nil {
		t.Fatalf("decode rename response: %v", err)
	}
	if renamed["name"] != "Renamed Product" {
		t.Fatalf("expected renamed product, got %+v", renamed["name"])
	}
	if renamed["code"] != originalCode {
		t.Fatalf("expected code preserved, got %+v", renamed["code"])
	}
	if renamed["description"] != description {
		t.Fatalf("expected description preserved, got %+v", renamed["description"])
	}

	clearDescResp := doJSON(t, handler, http.MethodPatch, "/api/v1/products/"+productID, map[string]any{
		"description": nil,
	}, cookies)
	if clearDescResp.Code != http.StatusOK {
		t.Fatalf("clear description status=%d body=%s", clearDescResp.Code, clearDescResp.Body.String())
	}
	var cleared map[string]any
	if err := json.Unmarshal(clearDescResp.Body.Bytes(), &cleared); err != nil {
		t.Fatalf("decode clear description response: %v", err)
	}
	if cleared["description"] != nil {
		t.Fatalf("expected description cleared, got %+v", cleared["description"])
	}
}

func TestPatchPolicyPartialUpdate(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productID, _ := createPatchTestProductAndPolicy(t, handler, cookies)
	description := "policy description"
	durationDays := 30
	gracePeriodDays := 7
	maxActivations := 2

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":        productID,
		"name":              "Patch Policy",
		"description":       description,
		"duration_days":     durationDays,
		"expiration_basis":  "on_first_validation",
		"grace_period_days": gracePeriodDays,
		"max_activations":   maxActivations,
	}, cookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	policyID := created["id"].(string)

	nameOnlyResp := doJSON(t, handler, http.MethodPatch, "/api/v1/policies/"+policyID, map[string]any{
		"name": "Renamed Policy",
	}, cookies)
	if nameOnlyResp.Code != http.StatusOK {
		t.Fatalf("patch policy name status=%d body=%s", nameOnlyResp.Code, nameOnlyResp.Body.String())
	}

	var renamed map[string]any
	if err := json.Unmarshal(nameOnlyResp.Body.Bytes(), &renamed); err != nil {
		t.Fatalf("decode rename response: %v", err)
	}
	if renamed["name"] != "Renamed Policy" {
		t.Fatalf("expected renamed policy, got %+v", renamed["name"])
	}
	if renamed["description"] != description {
		t.Fatalf("expected description preserved, got %+v", renamed["description"])
	}
	if int(renamed["duration_days"].(float64)) != durationDays {
		t.Fatalf("expected duration_days preserved, got %+v", renamed["duration_days"])
	}
	if renamed["expiration_basis"] != "on_first_validation" {
		t.Fatalf("expected expiration_basis preserved, got %+v", renamed["expiration_basis"])
	}
	if int(renamed["grace_period_days"].(float64)) != gracePeriodDays {
		t.Fatalf("expected grace_period_days preserved, got %+v", renamed["grace_period_days"])
	}
	if int(renamed["max_activations"].(float64)) != maxActivations {
		t.Fatalf("expected max_activations preserved, got %+v", renamed["max_activations"])
	}

	nullBasisResp := doJSON(t, handler, http.MethodPatch, "/api/v1/policies/"+policyID, map[string]any{
		"expiration_basis": nil,
	}, cookies)
	if nullBasisResp.Code != http.StatusBadRequest {
		t.Fatalf("null expiration_basis status=%d body=%s", nullBasisResp.Code, nullBasisResp.Body.String())
	}
}

func createPatchTestProductAndPolicy(t *testing.T, handler http.Handler, cookies []*http.Cookie) (string, string) {
	t.Helper()

	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Patch Test Product",
		"code": fmt.Sprintf("patch-test-%d", time.Now().UnixNano()),
	}, cookies)
	if productResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", productResp.Code, productResp.Body.String())
	}
	var product map[string]any
	if err := json.Unmarshal(productResp.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	policyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       product["id"],
		"name":             "Patch Test Policy",
		"expiration_basis": "on_creation",
	}, cookies)
	if policyResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", policyResp.Code, policyResp.Body.String())
	}
	var policy map[string]any
	if err := json.Unmarshal(policyResp.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy response: %v", err)
	}

	return product["id"].(string), policy["id"].(string)
}
