package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/license"
)

func TestAPIIntegration(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("test-product-%d", time.Now().UnixNano())

	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Test Product",
		"code": productCode,
	}, cookies)
	if productResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", productResp.Code, productResp.Body.String())
	}

	var product map[string]any
	if err := json.Unmarshal(productResp.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product response: %v", err)
	}
	productID := product["id"].(string)

	policyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       productID,
		"name":             "Perpetual",
		"expiration_basis": "on_creation",
	}, cookies)
	if policyResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", policyResp.Code, policyResp.Body.String())
	}

	var policy map[string]any
	if err := json.Unmarshal(policyResp.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy response: %v", err)
	}
	policyID := policy["id"].(string)

	createBody := map[string]any{
		"label":      "integration-test",
		"product_id": productID,
		"policy_id":  policyID,
	}
	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", createBody, cookies)
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

	licenseID, ok := created["id"].(string)
	if !ok || licenseID == "" {
		t.Fatalf("expected license id in create response")
	}

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses", nil, cookies)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list licenses status=%d", listResp.Code)
	}

	validateResp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key":     rawKey,
		"product": productCode,
	}, nil)
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
	if validation.Product != productCode {
		t.Fatalf("expected product %s, got %q", productCode, validation.Product)
	}

	mismatchResp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key":     rawKey,
		"product": "wrong-product",
	}, nil)
	var mismatchValidation license.ValidationResult
	if err := json.Unmarshal(mismatchResp.Body.Bytes(), &mismatchValidation); err != nil {
		t.Fatalf("decode mismatch validate response: %v", err)
	}
	if mismatchValidation.Valid || mismatchValidation.Reason != "product_mismatch" {
		t.Fatalf("expected product_mismatch, got %+v", mismatchValidation)
	}

	listAfterValidate := doJSON(t, handler, http.MethodGet, "/api/v1/licenses", nil, cookies)
	if listAfterValidate.Code != http.StatusOK {
		t.Fatalf("list licenses after validate status=%d", listAfterValidate.Code)
	}

	var licenses []map[string]any
	if err := json.Unmarshal(listAfterValidate.Body.Bytes(), &licenses); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	found := false
	for _, lic := range licenses {
		if lic["id"] == licenseID {
			found = true
			if count, ok := lic["validation_count"].(float64); !ok || count < 2 {
				t.Fatalf("expected validation_count >= 2, got %+v", lic["validation_count"])
			}
			if lic["last_validated_at"] == nil {
				t.Fatalf("expected last_validated_at to be set")
			}
		}
	}
	if !found {
		t.Fatalf("created license not found in list")
	}

	updateResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID, map[string]any{
		"label": "updated-label",
	}, cookies)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update license status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}

	var updated map[string]any
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated["label"] != "updated-label" {
		t.Fatalf("expected updated label, got %+v", updated["label"])
	}

	revokeResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID+"/revoke", nil, cookies)
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("revoke license status=%d", revokeResp.Code)
	}

	revokedValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": rawKey}, nil)
	var revokedValidation license.ValidationResult
	if err := json.Unmarshal(revokedValidate.Body.Bytes(), &revokedValidation); err != nil {
		t.Fatalf("decode revoked validate response: %v", err)
	}
	if revokedValidation.Valid || revokedValidation.Reason != "revoked" {
		t.Fatalf("expected revoked license, got %+v", revokedValidation)
	}

	activateResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID+"/activate", nil, cookies)
	if activateResp.Code != http.StatusOK {
		t.Fatalf("activate license status=%d", activateResp.Code)
	}

	reactivatedValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": rawKey}, nil)
	var reactivatedValidation license.ValidationResult
	if err := json.Unmarshal(reactivatedValidate.Body.Bytes(), &reactivatedValidation); err != nil {
		t.Fatalf("decode reactivated validate response: %v", err)
	}
	if !reactivatedValidation.Valid {
		t.Fatalf("expected valid license after activate, got %+v", reactivatedValidation)
	}

	deleteProductResp := doJSON(t, handler, http.MethodDelete, "/api/v1/products/"+productID, nil, cookies)
	if deleteProductResp.Code != http.StatusConflict {
		t.Fatalf("delete product with licenses status=%d body=%s", deleteProductResp.Code, deleteProductResp.Body.String())
	}

	deleteResp := doJSON(t, handler, http.MethodDelete, "/api/v1/licenses/"+licenseID, nil, cookies)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete license status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	deletedValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": rawKey}, nil)
	var deletedValidation license.ValidationResult
	if err := json.Unmarshal(deletedValidate.Body.Bytes(), &deletedValidation); err != nil {
		t.Fatalf("decode deleted validate response: %v", err)
	}
	if deletedValidation.Valid || deletedValidation.Reason != "not_found" {
		t.Fatalf("expected not_found after delete, got %+v", deletedValidation)
	}

	expiresAt := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	expiredResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "expired",
		"product_id": productID,
		"policy_id":  policyID,
		"expires_at": expiresAt,
	}, cookies)
	if expiredResp.Code != http.StatusCreated {
		t.Fatalf("create expired license status=%d", expiredResp.Code)
	}

	var expiredCreated map[string]any
	if err := json.Unmarshal(expiredResp.Body.Bytes(), &expiredCreated); err != nil {
		t.Fatalf("decode expired create response: %v", err)
	}

	expiredKey := expiredCreated["key"].(string)
	expiredValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": expiredKey}, nil)
	var expiredValidation license.ValidationResult
	if err := json.Unmarshal(expiredValidate.Body.Bytes(), &expiredValidation); err != nil {
		t.Fatalf("decode expired validate response: %v", err)
	}
	if expiredValidation.Valid || expiredValidation.Reason != "expired" {
		t.Fatalf("expected expired license, got %+v", expiredValidation)
	}

	trialPolicyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       productID,
		"name":             "Trial",
		"duration_days":    30,
		"expiration_basis": "on_first_validation",
	}, cookies)
	if trialPolicyResp.Code != http.StatusCreated {
		t.Fatalf("create trial policy status=%d body=%s", trialPolicyResp.Code, trialPolicyResp.Body.String())
	}

	var trialPolicy map[string]any
	if err := json.Unmarshal(trialPolicyResp.Body.Bytes(), &trialPolicy); err != nil {
		t.Fatalf("decode trial policy response: %v", err)
	}

	trialLicenseResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "trial-license",
		"product_id": productID,
		"policy_id":  trialPolicy["id"],
	}, cookies)
	if trialLicenseResp.Code != http.StatusCreated {
		t.Fatalf("create trial license status=%d body=%s", trialLicenseResp.Code, trialLicenseResp.Body.String())
	}

	var trialCreated map[string]any
	if err := json.Unmarshal(trialLicenseResp.Body.Bytes(), &trialCreated); err != nil {
		t.Fatalf("decode trial create response: %v", err)
	}
	if trialCreated["expires_at"] != nil {
		t.Fatalf("expected null expires_at before first validation, got %+v", trialCreated["expires_at"])
	}

	trialKey := trialCreated["key"].(string)
	trialValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": trialKey}, nil)
	var trialValidation license.ValidationResult
	if err := json.Unmarshal(trialValidate.Body.Bytes(), &trialValidation); err != nil {
		t.Fatalf("decode trial validate response: %v", err)
	}
	if !trialValidation.Valid {
		t.Fatalf("expected valid trial license, got %+v", trialValidation)
	}
	if trialValidation.ExpiresAt == nil {
		t.Fatalf("expected expires_at after first validation")
	}

	gracePolicyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":        productID,
		"name":              "Grace",
		"duration_days":     1,
		"expiration_basis":  "on_creation",
		"grace_period_days": 7,
	}, cookies)
	if gracePolicyResp.Code != http.StatusCreated {
		t.Fatalf("create grace policy status=%d body=%s", gracePolicyResp.Code, gracePolicyResp.Body.String())
	}

	var gracePolicy map[string]any
	if err := json.Unmarshal(gracePolicyResp.Body.Bytes(), &gracePolicy); err != nil {
		t.Fatalf("decode grace policy response: %v", err)
	}

	pastExpiry := time.Now().Add(-12 * time.Hour).UTC().Format(time.RFC3339)
	graceLicenseResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "grace-license",
		"product_id": productID,
		"policy_id":  gracePolicy["id"],
		"expires_at": pastExpiry,
	}, cookies)
	if graceLicenseResp.Code != http.StatusCreated {
		t.Fatalf("create grace license status=%d body=%s", graceLicenseResp.Code, graceLicenseResp.Body.String())
	}

	var graceCreated map[string]any
	if err := json.Unmarshal(graceLicenseResp.Body.Bytes(), &graceCreated); err != nil {
		t.Fatalf("decode grace create response: %v", err)
	}

	graceKey := graceCreated["key"].(string)
	graceValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": graceKey}, nil)
	var graceValidation license.ValidationResult
	if err := json.Unmarshal(graceValidate.Body.Bytes(), &graceValidation); err != nil {
		t.Fatalf("decode grace validate response: %v", err)
	}
	if !graceValidation.Valid || !graceValidation.InGracePeriod {
		t.Fatalf("expected valid license in grace period, got %+v", graceValidation)
	}

	listProductsResp := doJSON(t, handler, http.MethodGet, "/api/v1/products", nil, cookies)
	if listProductsResp.Code != http.StatusOK {
		t.Fatalf("list products status=%d", listProductsResp.Code)
	}

	listPoliciesResp := doJSON(t, handler, http.MethodGet, "/api/v1/policies?product_id="+productID, nil, cookies)
	if listPoliciesResp.Code != http.StatusOK {
		t.Fatalf("list policies status=%d", listPoliciesResp.Code)
	}
}
