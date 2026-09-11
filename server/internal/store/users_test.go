package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

func TestCountAdmins(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	before, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}

	hash, err := auth.HashPassword("count-admins-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	adminEmail := "count-admin-" + uuid.NewString() + "@example.com"
	admin, err := st.CreateUser(ctx, adminEmail, "Count Admin", &hash, store.RoleAdmin, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.DeleteUser(ctx, admin.ID)
	})

	afterCreate, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins after create: %v", err)
	}
	if afterCreate != before+1 {
		t.Fatalf("CountAdmins after create = %d, want %d", afterCreate, before+1)
	}

	if _, err := st.SetUserDisabled(ctx, admin.ID, true); err != nil {
		t.Fatalf("disable admin: %v", err)
	}
	afterDisable, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins after disable: %v", err)
	}
	if afterDisable != before {
		t.Fatalf("CountAdmins after disable = %d, want %d", afterDisable, before)
	}

	if _, err := st.SetUserDisabled(ctx, admin.ID, false); err != nil {
		t.Fatalf("enable admin: %v", err)
	}
	afterEnable, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins after enable: %v", err)
	}
	if afterEnable != before+1 {
		t.Fatalf("CountAdmins after enable = %d, want %d", afterEnable, before+1)
	}

	if _, err := st.UpdateUser(ctx, admin.ID, admin.Email, admin.Name, store.RoleOperator); err != nil {
		t.Fatalf("demote admin: %v", err)
	}
	afterDemote, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins after demote: %v", err)
	}
	if afterDemote != before {
		t.Fatalf("CountAdmins after demote = %d, want %d", afterDemote, before)
	}

	viewerEmail := "count-viewer-" + uuid.NewString() + "@example.com"
	viewer, err := st.CreateUser(ctx, viewerEmail, "Count Viewer", &hash, store.RoleViewer, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.DeleteUser(ctx, viewer.ID)
	})

	afterViewer, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins after viewer: %v", err)
	}
	if afterViewer != before {
		t.Fatalf("CountAdmins after viewer = %d, want %d", afterViewer, before)
	}
}

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
