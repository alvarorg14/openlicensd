package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
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

func TestListUsersPaginationAndSearch(t *testing.T) {
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
	hash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userA, err := st.CreateUser(ctx, "alpha-"+suffix+"@example.com", "Alpha User "+suffix, &hash, store.RoleViewer, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create user a: %v", err)
	}
	_, err = st.CreateUser(ctx, "beta-"+suffix+"@example.com", "Beta User "+suffix, &hash, store.RoleViewer, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create user b: %v", err)
	}

	users, total, err := st.ListUsers(ctx, store.ListParams{
		Search: suffix,
		Sort:   "name",
		Order:  "asc",
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if total < 2 {
		t.Fatalf("expected total >= 2, got %d", total)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].ID != userA.ID {
		t.Fatalf("expected first user by name sort to be alpha")
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
	policy, err := st.CreatePolicy(ctx, product.ID, "Policy", nil, nil, store.ExpirationOnCreation, 0, nil)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	activeLicense, err := st.CreateLicense(ctx, "active-"+suffix, "hash-"+suffix, "AAAA", product.ID, policy.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("create active license: %v", err)
	}
	revokedLicense, err := st.CreateLicense(ctx, "revoked-"+suffix, "hash-revoked-"+suffix, "BBBB", product.ID, policy.ID, nil, nil, nil)
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

func TestListLicensesActivationCount(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	suffix := uuid.NewString()
	product, err := st.CreateProduct(ctx, "Act Count Product "+suffix, "act-"+suffix, nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	policy, err := st.CreatePolicy(ctx, product.ID, "Policy", nil, nil, store.ExpirationOnCreation, 0, nil)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	licWithMachines, err := st.CreateLicense(ctx, "with-machines-"+suffix, "hash-wm-"+suffix, "WMCH1", product.ID, policy.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("create license with machines: %v", err)
	}
	licEmpty, err := st.CreateLicense(ctx, "empty-"+suffix, "hash-empty-"+suffix, "EMPT1", product.ID, policy.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("create empty license: %v", err)
	}

	for _, fp := range []string{"machine-a", "machine-b"} {
		_, allowed, err := st.RecordActivation(ctx, licWithMachines.ID, fp, "host-"+fp, "127.0.0.1", nil)
		if err != nil {
			t.Fatalf("RecordActivation %s: %v", fp, err)
		}
		if !allowed {
			t.Fatalf("expected activation %s to succeed", fp)
		}
	}

	licenses, _, err := st.ListLicenses(ctx, store.LicenseListParams{
		ListParams: store.ListParams{
			Search: suffix,
			Sort:   "l.created_at",
			Order:  "asc",
			Limit:  10,
			Offset: 0,
		},
	})
	if err != nil {
		t.Fatalf("list licenses: %v", err)
	}

	counts := make(map[uuid.UUID]int64, len(licenses))
	for _, lic := range licenses {
		counts[lic.ID] = lic.ActivationCount
	}
	if counts[licWithMachines.ID] != 2 {
		t.Fatalf("expected activation_count 2 for license with machines, got %d", counts[licWithMachines.ID])
	}
	if counts[licEmpty.ID] != 0 {
		t.Fatalf("expected activation_count 0 for empty license, got %d", counts[licEmpty.ID])
	}

	licenses, _, err = st.ListLicenses(ctx, store.LicenseListParams{
		ListParams: store.ListParams{
			Search: suffix,
			Sort:   "COALESCE(ac.activation_count, 0)",
			Order:  "desc",
			Limit:  10,
			Offset: 0,
		},
	})
	if err != nil {
		t.Fatalf("list licenses by activation_count: %v", err)
	}
	if len(licenses) < 2 || licenses[0].ID != licWithMachines.ID {
		t.Fatalf("expected license with machines first when sorting by activation_count desc")
	}

	machines, _, err := st.ListLicenseMachines(ctx, store.MachineListParams{
		ListParams: store.ListParams{
			Limit:  10,
			Offset: 0,
		},
		LicenseID: licWithMachines.ID,
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("list machines: %v", err)
	}
	if len(machines) == 0 {
		t.Fatal("expected at least one active machine")
	}

	if _, err := st.DeactivateMachine(ctx, licWithMachines.ID, machines[0].ID, nil); err != nil {
		t.Fatalf("deactivate machine: %v", err)
	}

	licenses, _, err = st.ListLicenses(ctx, store.LicenseListParams{
		ListParams: store.ListParams{
			Search: suffix,
			Sort:   "l.created_at",
			Order:  "asc",
			Limit:  10,
			Offset: 0,
		},
	})
	if err != nil {
		t.Fatalf("list licenses after deactivate: %v", err)
	}

	counts = make(map[uuid.UUID]int64, len(licenses))
	for _, lic := range licenses {
		counts[lic.ID] = lic.ActivationCount
	}
	if counts[licWithMachines.ID] != 1 {
		t.Fatalf("expected activation_count 1 after deactivate, got %d", counts[licWithMachines.ID])
	}

	byID, err := st.GetLicenseByID(ctx, licWithMachines.ID)
	if err != nil {
		t.Fatalf("get license by id: %v", err)
	}
	if byID == nil {
		t.Fatal("expected license by id")
	}
	if byID.ActivationCount != 1 {
		t.Fatalf("expected get-by-id activation_count 1, got %d", byID.ActivationCount)
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
	_, err = st.CreatePolicy(ctx, product.ID, "Named Policy "+suffix, nil, nil, store.ExpirationOnCreation, 0, nil)
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
