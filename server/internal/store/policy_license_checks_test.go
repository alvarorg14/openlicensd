package store

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPolicyNumericChecks(t *testing.T) {
	st := openIntegrationStore(t)
	ctx := context.Background()
	product := mustCreateCheckProduct(t, st, ctx)

	t.Run("grace_period_days negative", func(t *testing.T) {
		_, err := st.pool.Exec(ctx, `
			INSERT INTO policies (product_id, name, expiration_basis, grace_period_days)
			VALUES ($1, $2, 'on_creation', -1)
		`, product.ID, "grace-neg-"+uuid.NewString())
		if err == nil {
			t.Fatal("expected CHECK violation for negative grace_period_days")
		}
		if !isCheckViolation(err) {
			t.Fatalf("expected check violation, got %v", err)
		}
	})

	t.Run("duration_days zero", func(t *testing.T) {
		_, err := st.pool.Exec(ctx, `
			INSERT INTO policies (product_id, name, expiration_basis, duration_days)
			VALUES ($1, $2, 'on_creation', 0)
		`, product.ID, "duration-zero-"+uuid.NewString())
		if err == nil {
			t.Fatal("expected CHECK violation for duration_days = 0")
		}
		if !isCheckViolation(err) {
			t.Fatalf("expected check violation, got %v", err)
		}
	})

	t.Run("max_activations zero", func(t *testing.T) {
		_, err := st.pool.Exec(ctx, `
			INSERT INTO policies (product_id, name, expiration_basis, max_activations)
			VALUES ($1, $2, 'on_creation', 0)
		`, product.ID, "max-zero-"+uuid.NewString())
		if err == nil {
			t.Fatal("expected CHECK violation for max_activations = 0")
		}
		if !isCheckViolation(err) {
			t.Fatalf("expected check violation, got %v", err)
		}
	})
}

func TestLicenseNumericChecks(t *testing.T) {
	st := openIntegrationStore(t)
	ctx := context.Background()
	product := mustCreateCheckProduct(t, st, ctx)
	policy := mustCreateCheckPolicy(t, st, ctx, product.ID)

	t.Run("max_activations zero", func(t *testing.T) {
		_, err := st.pool.Exec(ctx, `
			INSERT INTO licenses (label, key_hash, key_prefix, product_id, policy_id, max_activations)
			VALUES ($1, $2, 'XXXXX', $3, $4, 0)
		`, "check-license", "hash-max-"+uuid.NewString(), product.ID, policy.ID)
		if err == nil {
			t.Fatal("expected CHECK violation for licenses.max_activations = 0")
		}
		if !isCheckViolation(err) {
			t.Fatalf("expected check violation, got %v", err)
		}
	})

	t.Run("validation_count negative", func(t *testing.T) {
		_, err := st.pool.Exec(ctx, `
			INSERT INTO licenses (label, key_hash, key_prefix, product_id, policy_id, validation_count)
			VALUES ($1, $2, 'XXXXX', $3, $4, -1)
		`, "check-license", "hash-count-"+uuid.NewString(), product.ID, policy.ID)
		if err == nil {
			t.Fatal("expected CHECK violation for licenses.validation_count = -1")
		}
		if !isCheckViolation(err) {
			t.Fatalf("expected check violation, got %v", err)
		}
	})
}

func TestLicenseMachineValidationCountCheck(t *testing.T) {
	st := openIntegrationStore(t)
	ctx := context.Background()
	product := mustCreateCheckProduct(t, st, ctx)
	policy := mustCreateCheckPolicy(t, st, ctx, product.ID)
	lic, err := st.CreateLicense(ctx, "machine-check", "hash-machine-"+uuid.NewString(), "XXXXX", product.ID, policy.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("create license: %v", err)
	}

	_, err = st.pool.Exec(ctx, `
		INSERT INTO license_machines (license_id, fingerprint, validation_count)
		VALUES ($1, $2, -1)
	`, lic.ID, "fp-"+uuid.NewString())
	if err == nil {
		t.Fatal("expected CHECK violation for license_machines.validation_count = -1")
	}
	if !isCheckViolation(err) {
		t.Fatalf("expected check violation, got %v", err)
	}
}

func mustCreateCheckProduct(t *testing.T, st *Store, ctx context.Context) *Product {
	t.Helper()
	product, err := st.CreateProduct(ctx, "Check Product", uuid.NewString(), nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	return product
}

func mustCreateCheckPolicy(t *testing.T, st *Store, ctx context.Context, productID uuid.UUID) *Policy {
	t.Helper()
	policy, err := st.CreatePolicy(ctx, productID, "Check Policy "+uuid.NewString(), nil, nil, ExpirationOnCreation, 0, nil)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	return policy
}

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23514") || strings.Contains(msg, "check constraint")
}
