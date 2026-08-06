package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

func TestOIDCUserLookupAndLink(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)

	externalID := "oidc-sub-" + uuid.NewString()
	email := "oidc-link-" + uuid.NewString() + "@example.com"

	hash, err := auth.HashPassword("local-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	localUser, err := st.CreateUser(ctx, email, "Local User", &hash, store.RoleOperator, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create local user: %v", err)
	}

	linked, err := st.LinkUserToProvider(ctx, localUser.ID, store.AuthProviderOIDC, externalID)
	if err != nil {
		t.Fatalf("link user: %v", err)
	}
	if linked.AuthProvider != store.AuthProviderOIDC {
		t.Fatalf("auth_provider=%q want oidc", linked.AuthProvider)
	}
	if linked.ExternalID == nil || *linked.ExternalID != externalID {
		t.Fatalf("external_id=%v", linked.ExternalID)
	}
	if linked.PasswordHash == nil || *linked.PasswordHash == "" {
		t.Fatalf("expected password hash preserved after link")
	}
	if linked.Role != store.RoleOperator {
		t.Fatalf("role=%q want operator preserved", linked.Role)
	}

	byExternal, err := st.GetUserByExternalID(ctx, store.AuthProviderOIDC, externalID)
	if err != nil {
		t.Fatalf("get by external id: %v", err)
	}
	if byExternal == nil || byExternal.ID != localUser.ID {
		t.Fatalf("expected linked user by external id")
	}

	updated, err := st.SyncUserProfile(ctx, localUser.ID, email, "Updated Name", nil)
	if err != nil {
		t.Fatalf("sync profile: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Fatalf("name=%q", updated.Name)
	}
	if updated.Role != store.RoleOperator {
		t.Fatalf("role changed after sync profile")
	}
}
