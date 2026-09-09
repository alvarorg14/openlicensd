package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

func TestUpdateProductPartialPatch(t *testing.T) {
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

	desc := "keep me"
	product, err := st.CreateProduct(ctx, "Patch Product", uuid.NewString(), &desc)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	newName := "Renamed"
	updated, err := st.UpdateProduct(ctx, product.ID, store.ProductPatch{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}
	if updated.Code != product.Code {
		t.Fatalf("expected code preserved, got %q", updated.Code)
	}
	if updated.Description == nil || *updated.Description != desc {
		t.Fatalf("expected description preserved, got %+v", updated.Description)
	}
}

func TestUpdatePolicyPartialPatch(t *testing.T) {
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

	product, err := st.CreateProduct(ctx, "Patch Policy Product", uuid.NewString(), nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	desc := "policy desc"
	duration := 14
	grace := 3
	maxActivations := 4
	policy, err := st.CreatePolicy(
		ctx,
		product.ID,
		"Original Policy",
		&desc,
		&duration,
		store.ExpirationOnFirstValidation,
		grace,
		&maxActivations,
	)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	newName := "Renamed Policy"
	updated, err := st.UpdatePolicy(ctx, policy.ID, store.PolicyPatch{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}
	if updated.Description == nil || *updated.Description != desc {
		t.Fatalf("expected description preserved, got %+v", updated.Description)
	}
	if updated.DurationDays == nil || *updated.DurationDays != duration {
		t.Fatalf("expected duration_days preserved, got %+v", updated.DurationDays)
	}
	if updated.ExpirationBasis != store.ExpirationOnFirstValidation {
		t.Fatalf("expected expiration_basis preserved, got %q", updated.ExpirationBasis)
	}
	if updated.GracePeriodDays != grace {
		t.Fatalf("expected grace_period_days preserved, got %d", updated.GracePeriodDays)
	}
	if updated.MaxActivations == nil || *updated.MaxActivations != maxActivations {
		t.Fatalf("expected max_activations preserved, got %+v", updated.MaxActivations)
	}
}

func TestUpdateLicensePartialPatch(t *testing.T) {
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

	product, err := st.CreateProduct(ctx, "Patch License Product", uuid.NewString(), nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	policy, err := st.CreatePolicy(ctx, product.ID, "Policy", nil, nil, store.ExpirationOnCreation, 0, nil)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	maxActivations := 2
	lic, err := st.CreateLicense(ctx, "Original", "hash", "prefix", product.ID, policy.ID, nil, &maxActivations, nil)
	if err != nil {
		t.Fatalf("create license: %v", err)
	}

	newLabel := "Renamed License"
	updated, err := st.UpdateLicense(ctx, lic.ID, store.LicensePatch{
		Label: &newLabel,
	})
	if err != nil {
		t.Fatalf("update license: %v", err)
	}
	if updated.Label != newLabel {
		t.Fatalf("expected label %q, got %q", newLabel, updated.Label)
	}
	if updated.MaxActivations == nil || *updated.MaxActivations != maxActivations {
		t.Fatalf("expected max_activations preserved, got %+v", updated.MaxActivations)
	}
}
