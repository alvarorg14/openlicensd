package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestListEndpointsPagination(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("list-product-%d", time.Now().UnixNano())
	productResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "List Product",
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
		"name":             "List Policy",
		"expiration_basis": "on_creation",
	}, cookies)
	if policyResp.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", policyResp.Code, policyResp.Body.String())
	}

	var policy map[string]any
	if err := json.Unmarshal(policyResp.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	if policy["product_name"] != "List Product" {
		t.Fatalf("expected product_name on policy, got %+v", policy["product_name"])
	}
	policyID := policy["id"].(string)

	searchLabel := fmt.Sprintf("license-search-%d", time.Now().UnixNano())
	for i := range 3 {
		label := fmt.Sprintf("license-%d", i)
		if i == 0 {
			label = searchLabel
		}
		createResp := doJSON(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
			"label":      label,
			"product_id": productID,
			"policy_id":  policyID,
		}, cookies)
		if createResp.Code != http.StatusCreated {
			t.Fatalf("create license %d status=%d", i, createResp.Code)
		}
	}

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses?page=1&page_size=2", nil, cookies)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list licenses status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	var page map[string]any
	if err := json.Unmarshal(listResp.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode list page: %v", err)
	}

	if page["page"].(float64) != 1 {
		t.Fatalf("expected page=1, got %+v", page["page"])
	}
	if page["page_size"].(float64) != 2 {
		t.Fatalf("expected page_size=2, got %+v", page["page_size"])
	}
	if page["total"].(float64) < 3 {
		t.Fatalf("expected total >= 3, got %+v", page["total"])
	}
	items, ok := page["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 items, got %+v", page["items"])
	}

	searchResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses?search="+searchLabel, nil, cookies)
	if searchResp.Code != http.StatusOK {
		t.Fatalf("search licenses status=%d", searchResp.Code)
	}
	var searchPage map[string]any
	if err := json.Unmarshal(searchResp.Body.Bytes(), &searchPage); err != nil {
		t.Fatalf("decode search page: %v", err)
	}
	if searchPage["total"].(float64) != 1 {
		t.Fatalf("expected search total=1, got %+v", searchPage["total"])
	}

	statusResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses?status=active", nil, cookies)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("status filter status=%d", statusResp.Code)
	}

	badSortResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses?sort=invalid", nil, cookies)
	if badSortResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid sort, got %d", badSortResp.Code)
	}

	badPageResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses?page=0", nil, cookies)
	if badPageResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid page, got %d", badPageResp.Code)
	}

	statsResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses/stats", nil, cookies)
	if statsResp.Code != http.StatusOK {
		t.Fatalf("license stats status=%d body=%s", statsResp.Code, statsResp.Body.String())
	}
	var stats map[string]any
	if err := json.Unmarshal(statsResp.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats["total"].(float64) < 3 {
		t.Fatalf("expected stats total >= 3, got %+v", stats["total"])
	}

	productsResp := doJSON(t, handler, http.MethodGet, "/api/v1/products?search=List", nil, cookies)
	if productsResp.Code != http.StatusOK {
		t.Fatalf("list products status=%d", productsResp.Code)
	}
	var productsPage map[string]any
	if err := json.Unmarshal(productsResp.Body.Bytes(), &productsPage); err != nil {
		t.Fatalf("decode products page: %v", err)
	}
	if productsPage["items"] == nil {
		t.Fatalf("expected products items")
	}

	policiesResp := doJSON(t, handler, http.MethodGet, "/api/v1/policies?product_id="+productID, nil, cookies)
	if policiesResp.Code != http.StatusOK {
		t.Fatalf("list policies status=%d", policiesResp.Code)
	}
	var policiesPage map[string]any
	if err := json.Unmarshal(policiesResp.Body.Bytes(), &policiesPage); err != nil {
		t.Fatalf("decode policies page: %v", err)
	}
	policyItems, ok := policiesPage["items"].([]any)
	if !ok || len(policyItems) == 0 {
		t.Fatalf("expected policy items")
	}
}
