package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/license"
)

func TestHealthzSecurityHeaders(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	env.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want %q", got, "DENY")
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("Content-Security-Policy is empty")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("Strict-Transport-Security = %q, want empty when CookieSecure is false", got)
	}
}

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

	var listPage map[string]any
	if err := json.Unmarshal(listAfterValidate.Body.Bytes(), &listPage); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	licenses, ok := listPage["items"].([]any)
	if !ok {
		t.Fatalf("expected items array in list response, got %+v", listPage)
	}

	found := false
	for _, lic := range licenses {
		item := lic.(map[string]any)
		if item["id"] == licenseID {
			found = true
			if count, ok := item["validation_count"].(float64); !ok || count != 1 {
				t.Fatalf("expected validation_count == 1, got %+v", item["validation_count"])
			}
			if item["last_validated_at"] == nil {
				t.Fatalf("expected last_validated_at to be set")
			}
		}
	}
	if !found {
		t.Fatalf("created license not found in list")
	}

	getResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+licenseID, nil, cookies)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get license status=%d body=%s", getResp.Code, getResp.Body.String())
	}
	var fetched map[string]any
	if err := json.Unmarshal(getResp.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get license response: %v", err)
	}
	if fetched["id"] != licenseID {
		t.Fatalf("expected license id %s, got %+v", licenseID, fetched["id"])
	}
	if fetched["label"] != "integration-test" {
		t.Fatalf("expected label integration-test, got %+v", fetched["label"])
	}
	if fetched["product_id"] != productID {
		t.Fatalf("expected product_id %s, got %+v", productID, fetched["product_id"])
	}
	if _, hasKey := fetched["key"]; hasKey {
		t.Fatalf("expected key to be omitted from get response")
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

	revokedGetResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+licenseID, nil, cookies)
	if revokedGetResp.Code != http.StatusOK {
		t.Fatalf("get revoked license status=%d body=%s", revokedGetResp.Code, revokedGetResp.Body.String())
	}
	var revokedFetched map[string]any
	if err := json.Unmarshal(revokedGetResp.Body.Bytes(), &revokedFetched); err != nil {
		t.Fatalf("decode revoked get license response: %v", err)
	}
	if revoked, ok := revokedFetched["revoked"].(bool); !ok || !revoked {
		t.Fatalf("expected revoked license, got %+v", revokedFetched["revoked"])
	}

	revokedValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": rawKey}, nil)
	var revokedValidation license.ValidationResult
	if err := json.Unmarshal(revokedValidate.Body.Bytes(), &revokedValidation); err != nil {
		t.Fatalf("decode revoked validate response: %v", err)
	}
	if revokedValidation.Valid || revokedValidation.Reason != "revoked" {
		t.Fatalf("expected revoked license, got %+v", revokedValidation)
	}

	unrevokeResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID+"/unrevoke", nil, cookies)
	if unrevokeResp.Code != http.StatusOK {
		t.Fatalf("unrevoke license status=%d", unrevokeResp.Code)
	}

	removedActivateResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID+"/activate", nil, cookies)
	if removedActivateResp.Code != http.StatusNotFound {
		t.Fatalf("expected removed activate endpoint status=404, got %d", removedActivateResp.Code)
	}

	reactivatedValidate := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{"key": rawKey}, nil)
	var reactivatedValidation license.ValidationResult
	if err := json.Unmarshal(reactivatedValidate.Body.Bytes(), &reactivatedValidation); err != nil {
		t.Fatalf("decode reactivated validate response: %v", err)
	}
	if !reactivatedValidation.Valid {
		t.Fatalf("expected valid license after unrevoke, got %+v", reactivatedValidation)
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

func getLicenseValidationCount(t *testing.T, handler http.Handler, licenseID string, cookies []*http.Cookie) int64 {
	t.Helper()
	resp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+licenseID, nil, cookies)
	if resp.Code != http.StatusOK {
		t.Fatalf("get license status=%d body=%s", resp.Code, resp.Body.String())
	}
	var lic map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &lic); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	count, ok := lic["validation_count"].(float64)
	if !ok {
		t.Fatalf("expected validation_count number, got %+v", lic["validation_count"])
	}
	return int64(count)
}

func TestValidationCountOnlyOnSuccess(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("validation-count-product-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Validation Count Product",
		"code": productCode,
	}, cookies)
	if productResp.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", productResp.Code, productResp.Body.String())
	}
	var product map[string]any
	if err := json.Unmarshal(productResp.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product: %v", err)
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
		t.Fatalf("decode policy: %v", err)
	}

	licenseResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "validation-count-license",
		"product_id": productID,
		"policy_id":  policy["id"],
	}, cookies)
	if licenseResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", licenseResp.Code, licenseResp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(licenseResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	rawKey := created["key"].(string)
	licenseID := created["id"].(string)

	validate := func(body map[string]string) license.ValidationResult {
		t.Helper()
		resp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", body, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("validate status=%d body=%s", resp.Code, resp.Body.String())
		}
		var result license.ValidationResult
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode validate: %v", err)
		}
		return result
	}

	assertCount := func(want int64) {
		t.Helper()
		if got := getLicenseValidationCount(t, handler, licenseID, cookies); got != want {
			t.Fatalf("expected validation_count %d, got %d", want, got)
		}
	}

	result := validate(map[string]string{"key": rawKey, "product": productCode})
	if !result.Valid {
		t.Fatalf("expected valid license, got %+v", result)
	}
	assertCount(1)

	mismatch := validate(map[string]string{"key": rawKey, "product": "wrong-product"})
	if mismatch.Valid || mismatch.Reason != "product_mismatch" {
		t.Fatalf("expected product_mismatch, got %+v", mismatch)
	}
	assertCount(1)

	revokeResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID+"/revoke", nil, cookies)
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("revoke license status=%d", revokeResp.Code)
	}
	revoked := validate(map[string]string{"key": rawKey})
	if revoked.Valid || revoked.Reason != "revoked" {
		t.Fatalf("expected revoked, got %+v", revoked)
	}
	assertCount(1)

	unrevokeResp := doJSON(t, handler, http.MethodPatch, "/api/v1/licenses/"+licenseID+"/unrevoke", nil, cookies)
	if unrevokeResp.Code != http.StatusOK {
		t.Fatalf("unrevoke license status=%d", unrevokeResp.Code)
	}
	reactivated := validate(map[string]string{"key": rawKey, "product": productCode})
	if !reactivated.Valid {
		t.Fatalf("expected valid after unrevoke, got %+v", reactivated)
	}
	assertCount(2)

	expiredResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "expired-validation-count",
		"product_id": productID,
		"policy_id":  policy["id"],
		"expires_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
	}, cookies)
	if expiredResp.Code != http.StatusCreated {
		t.Fatalf("create expired license status=%d", expiredResp.Code)
	}
	var expiredCreated map[string]any
	if err := json.Unmarshal(expiredResp.Body.Bytes(), &expiredCreated); err != nil {
		t.Fatalf("decode expired license: %v", err)
	}
	expiredKey := expiredCreated["key"].(string)
	expiredID := expiredCreated["id"].(string)

	expired := validate(map[string]string{"key": expiredKey})
	if expired.Valid || expired.Reason != "expired" {
		t.Fatalf("expected expired, got %+v", expired)
	}
	if got := getLicenseValidationCount(t, handler, expiredID, cookies); got != 0 {
		t.Fatalf("expected expired license validation_count 0, got %d", got)
	}

	fpPolicyResp := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       productID,
		"name":             "One seat",
		"expiration_basis": "on_creation",
		"max_activations":  1,
	}, cookies)
	if fpPolicyResp.Code != http.StatusCreated {
		t.Fatalf("create fingerprint policy status=%d body=%s", fpPolicyResp.Code, fpPolicyResp.Body.String())
	}
	var fpPolicy map[string]any
	if err := json.Unmarshal(fpPolicyResp.Body.Bytes(), &fpPolicy); err != nil {
		t.Fatalf("decode fingerprint policy: %v", err)
	}

	fpLicenseResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "fingerprint-required-license",
		"product_id": productID,
		"policy_id":  fpPolicy["id"],
	}, cookies)
	if fpLicenseResp.Code != http.StatusCreated {
		t.Fatalf("create fingerprint license status=%d body=%s", fpLicenseResp.Code, fpLicenseResp.Body.String())
	}
	var fpCreated map[string]any
	if err := json.Unmarshal(fpLicenseResp.Body.Bytes(), &fpCreated); err != nil {
		t.Fatalf("decode fingerprint license: %v", err)
	}
	fpKey := fpCreated["key"].(string)
	fpLicenseID := fpCreated["id"].(string)

	noFp := validate(map[string]string{"key": fpKey, "product": productCode})
	if noFp.Valid || noFp.Reason != "fingerprint_required" {
		t.Fatalf("expected fingerprint_required, got %+v", noFp)
	}
	if got := getLicenseValidationCount(t, handler, fpLicenseID, cookies); got != 0 {
		t.Fatalf("expected fingerprint-required license validation_count 0, got %d", got)
	}
}

func TestActivationLimits(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("activation-product-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Activation Product",
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
		"name":             "Two seats",
		"expiration_basis": "on_creation",
		"max_activations":  2,
	}, cookies)
	if policyResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", policyResp.Code, policyResp.Body.String())
	}
	var policy map[string]any
	if err := json.Unmarshal(policyResp.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy: %v", err)
	}

	licenseResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "activation-license",
		"product_id": product["id"],
		"policy_id":  policy["id"],
	}, cookies)
	if licenseResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", licenseResp.Code, licenseResp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(licenseResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	rawKey := created["key"].(string)
	licenseID := created["id"].(string)

	noFpResp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key":     rawKey,
		"product": productCode,
	}, nil)
	var noFp license.ValidationResult
	if err := json.Unmarshal(noFpResp.Body.Bytes(), &noFp); err != nil {
		t.Fatalf("decode validate without fingerprint: %v", err)
	}
	if noFp.Valid || noFp.Reason != "fingerprint_required" {
		t.Fatalf("expected fingerprint_required, got %+v", noFp)
	}

	for _, fp := range []string{"machine-a", "machine-b"} {
		resp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
			"key":         rawKey,
			"product":     productCode,
			"fingerprint": fp,
			"hostname":    fp + ".local",
		}, nil)
		var result license.ValidationResult
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode validate: %v", err)
		}
		if !result.Valid {
			t.Fatalf("expected valid activation for %s, got %+v", fp, result)
		}
	}

	limitResp := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key":         rawKey,
		"product":     productCode,
		"fingerprint": "machine-c",
	}, nil)
	var limit license.ValidationResult
	if err := json.Unmarshal(limitResp.Body.Bytes(), &limit); err != nil {
		t.Fatalf("decode limit validate: %v", err)
	}
	if limit.Valid || limit.Reason != "activation_limit" {
		t.Fatalf("expected activation_limit, got %+v", limit)
	}

	machinesResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+licenseID+"/machines", nil, cookies)
	if machinesResp.Code != http.StatusOK {
		t.Fatalf("list machines status=%d body=%s", machinesResp.Code, machinesResp.Body.String())
	}
	var machinesPage map[string]any
	if err := json.Unmarshal(machinesResp.Body.Bytes(), &machinesPage); err != nil {
		t.Fatalf("decode machines: %v", err)
	}
	items, ok := machinesPage["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 machines, got %+v", machinesPage)
	}

	firstMachine := items[0].(map[string]any)
	machineID := firstMachine["id"].(string)

	releaseResp := doJSON(t, handler, http.MethodDelete, "/api/v1/licenses/"+licenseID+"/machines/"+machineID, nil, cookies)
	if releaseResp.Code != http.StatusOK {
		t.Fatalf("release machine status=%d body=%s", releaseResp.Code, releaseResp.Body.Bytes())
	}

	afterRelease := doJSON(t, handler, http.MethodPost, "/api/v1/validate", map[string]string{
		"key":         rawKey,
		"product":     productCode,
		"fingerprint": "machine-c",
	}, nil)
	var after license.ValidationResult
	if err := json.Unmarshal(afterRelease.Body.Bytes(), &after); err != nil {
		t.Fatalf("decode after release: %v", err)
	}
	if !after.Valid {
		t.Fatalf("expected valid activation after release, got %+v", after)
	}
}

func TestGetLicense(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("get-license-product-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Get License Product",
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

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "get-license-test",
		"product_id": productID,
		"policy_id":  policyID,
	}, cookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create license status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	licenseID := created["id"].(string)

	badIDResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/not-a-uuid", nil, cookies)
	if badIDResp.Code != http.StatusBadRequest {
		t.Fatalf("invalid license id status=%d want 400", badIDResp.Code)
	}

	missingID := "00000000-0000-0000-0000-000000000000"
	notFoundResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+missingID, nil, cookies)
	if notFoundResp.Code != http.StatusNotFound {
		t.Fatalf("missing license status=%d want 404", notFoundResp.Code)
	}

	unauthResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+licenseID, nil, nil)
	if unauthResp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated get license status=%d want 401", unauthResp.Code)
	}

	machinesNotFoundResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/"+missingID+"/machines", nil, cookies)
	if machinesNotFoundResp.Code != http.StatusNotFound {
		t.Fatalf("missing license machines status=%d want 404", machinesNotFoundResp.Code)
	}
}
