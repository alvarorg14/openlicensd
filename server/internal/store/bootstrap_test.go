package store_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

func TestBootstrapAdminNoOpWhenUsersExist(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	count, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count == 0 {
		t.Skip("empty database required for opposite cases; users already exist here")
	}

	before := count
	cfg := &config.Config{
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        "bootstrap-noop-" + uuid.NewString() + "@example.com",
			Name:         "Bootstrap",
			PasswordHash: "not-used",
		},
	}

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}

	after, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers after: %v", err)
	}
	if after != before {
		t.Fatalf("CountUsers = %d, want %d (no new user)", after, before)
	}
}

func TestBootstrapAdminConcurrent(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	count, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}

	hash, err := auth.HashPassword("bootstrap-concurrent-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := "bootstrap-concurrent-" + uuid.NewString() + "@example.com"
	cfg := &config.Config{
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        email,
			Name:         "Bootstrap Admin",
			PasswordHash: hash,
		},
	}

	before := count
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = store.BootstrapAdmin(ctx, st, cfg)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	after, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers after: %v", err)
	}

	if before == 0 {
		if after != 1 {
			t.Fatalf("CountUsers = %d, want 1 after seeding empty DB", after)
		}
		user, err := st.GetUserByEmail(ctx, email)
		if err != nil {
			t.Fatalf("GetUserByEmail: %v", err)
		}
		if user == nil {
			t.Fatal("expected bootstrap user")
		}
		t.Cleanup(func() {
			_, _ = st.DeleteUser(ctx, user.ID)
		})
	} else if after != before {
		t.Fatalf("CountUsers = %d, want %d (no new user when DB not empty)", after, before)
	}
}

func TestBootstrapAdminSeedsEmptyDB(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	count, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count > 0 {
		t.Skip("database not empty")
	}

	hash, err := auth.HashPassword("bootstrap-seed-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := "bootstrap-seed-" + uuid.NewString() + "@example.com"
	cfg := &config.Config{
		BootstrapAdmin: config.BootstrapAdminConfig{
			Email:        email,
			Name:         "Seed Admin",
			PasswordHash: hash,
		},
	}

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		t.Fatalf("first BootstrapAdmin: %v", err)
	}

	user, err := st.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if user == nil {
		t.Fatal("expected seeded user")
	}
	t.Cleanup(func() {
		_, _ = st.DeleteUser(ctx, user.ID)
	})

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		t.Fatalf("second BootstrapAdmin: %v", err)
	}

	after, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers after second call: %v", err)
	}
	if after != 1 {
		t.Fatalf("CountUsers = %d, want 1", after)
	}
}

func TestBootstrapAdminMissingEnvOnEmptyDB(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	count, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count > 0 {
		t.Skip("database not empty")
	}

	cfg := &config.Config{
		BootstrapAdmin: config.BootstrapAdminConfig{},
	}

	err = store.BootstrapAdmin(ctx, st, cfg)
	if err == nil {
		t.Fatal("expected error when bootstrap env missing on empty DB")
	}
	if !strings.Contains(err.Error(), "no users exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}
