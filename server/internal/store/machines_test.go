package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

func TestRecordActivationEnforcesLimit(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	suffix := uuid.NewString()
	product, err := st.CreateProduct(ctx, "Activation Product", "activation-"+suffix, nil)
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	max := 2
	policy, err := st.CreatePolicy(ctx, product.ID, "Limited", nil, nil, store.ExpirationOnCreation, 0, &max)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	lic, err := st.CreateLicense(ctx, "limited-license", "hash-limited-"+suffix, "LMITD", product.ID, policy.ID, nil, &max, nil)
	if err != nil {
		t.Fatalf("CreateLicense: %v", err)
	}

	for i, fp := range []string{"machine-a", "machine-b", "machine-c"} {
		_, allowed, err := st.RecordActivation(ctx, lic.ID, fp, "host-"+fp, "127.0.0.1", &max)
		if err != nil {
			t.Fatalf("RecordActivation %d: %v", i, err)
		}
		if i < 2 && !allowed {
			t.Fatalf("expected activation %d to succeed", i)
		}
		if i == 2 && allowed {
			t.Fatal("expected third activation to be rejected")
		}
	}

	count, err := st.CountActiveMachines(ctx, lic.ID)
	if err != nil {
		t.Fatalf("CountActiveMachines: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 active machines, got %d", count)
	}
}

func TestRecordActivationReusesKnownFingerprint(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	suffix := uuid.NewString()
	product, err := st.CreateProduct(ctx, "Reuse Product", "reuse-"+suffix, nil)
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	max := 1
	policy, err := st.CreatePolicy(ctx, product.ID, "Single Seat", nil, nil, store.ExpirationOnCreation, 0, &max)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	lic, err := st.CreateLicense(ctx, "single-seat", "hash-single-"+suffix, "SEAT1", product.ID, policy.ID, nil, &max, nil)
	if err != nil {
		t.Fatalf("CreateLicense: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, allowed, err := st.RecordActivation(ctx, lic.ID, "same-machine", "host-a", "127.0.0.1", &max)
		if err != nil {
			t.Fatalf("RecordActivation %d: %v", i, err)
		}
		if !allowed {
			t.Fatalf("expected known fingerprint to remain allowed on attempt %d", i)
		}
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}
