package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

func TestListProductsPaginationAndSearch(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, databaseURL)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(st.Close)

	suffix := uuid.NewString()
	productA, err := st.CreateProduct(ctx, "Alpha Widget "+suffix, "alpha-"+suffix, nil)
	if err != nil {
		t.Fatalf("create product a: %v", err)
	}
	_, err = st.CreateProduct(ctx, "Beta Gadget "+suffix, "beta-"+suffix, nil)
	if err != nil {
		t.Fatalf("create product b: %v", err)
	}

	products, total, err := st.ListProducts(ctx, store.ListParams{
		Search: suffix,
		Sort:   "name",
		Order:  "asc",
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	if total < 2 {
		t.Fatalf("expected total >= 2, got %d", total)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].ID != productA.ID {
		t.Fatalf("expected first product by name sort to be alpha")
	}
}

func TestListLicensesStatusAndStats(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, databaseURL)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(st.Close)

	suffix := uuid.NewString()
	product, err := st.CreateProduct(ctx, "License List "+suffix, "lic-"+suffix, nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	policy, err := st.CreatePolicy(ctx, product.ID, "Policy", nil, nil, store.ExpirationOnCreation, 0)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	activeLicense, err := st.CreateLicense(ctx, "active-"+suffix, "hash-"+suffix, "AAAA", product.ID, policy.ID, nil, nil)
	if err != nil {
		t.Fatalf("create active license: %v", err)
	}
	revokedLicense, err := st.CreateLicense(ctx, "revoked-"+suffix, "hash-revoked-"+suffix, "BBBB", product.ID, policy.ID, nil, nil)
	if err != nil {
		t.Fatalf("create revoked license: %v", err)
	}
	if _, err := st.SetLicenseRevoked(ctx, revokedLicense.ID, true); err != nil {
		t.Fatalf("revoke license: %v", err)
	}

	status := "revoked"
	licenses, total, err := st.ListLicenses(ctx, store.LicenseListParams{
		ListParams: store.ListParams{
			Search: suffix,
			Sort:   "l.created_at",
			Order:  "desc",
			Limit:  10,
			Offset: 0,
		},
		Status: status,
	})
	if err != nil {
		t.Fatalf("list revoked licenses: %v", err)
	}
	if total < 1 {
		t.Fatalf("expected at least one revoked license")
	}
	if len(licenses) == 0 || !licenses[0].Revoked {
		t.Fatalf("expected revoked license in results")
	}
	_ = activeLicense

	stats, err := st.LicenseStats(ctx)
	if err != nil {
		t.Fatalf("license stats: %v", err)
	}
	if stats.Total < 2 {
		t.Fatalf("expected stats total >= 2, got %d", stats.Total)
	}
	if stats.Revoked < 1 {
		t.Fatalf("expected revoked count >= 1, got %d", stats.Revoked)
	}
}

func TestListPoliciesWithProductName(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, databaseURL)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(st.Close)

	suffix := uuid.NewString()
	product, err := st.CreateProduct(ctx, "Policy Product "+suffix, "pol-"+suffix, nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	_, err = st.CreatePolicy(ctx, product.ID, "Named Policy "+suffix, nil, nil, store.ExpirationOnCreation, 0)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	productID := product.ID.String()
	policies, total, err := st.ListPolicies(ctx, store.PolicyListParams{
		ListParams: store.ListParams{
			Search: suffix,
			Sort:   "pol.name",
			Order:  "asc",
			Limit:  10,
			Offset: 0,
		},
		ProductID: &productID,
	})
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if total < 1 || len(policies) == 0 {
		t.Fatalf("expected policies")
	}
	if policies[0].ProductName != product.Name {
		t.Fatalf("expected product name %q, got %q", product.Name, policies[0].ProductName)
	}
}
