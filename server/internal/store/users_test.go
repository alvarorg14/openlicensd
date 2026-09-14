package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/config"
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

func TestLastAdminGuardStoreSoleAdmin(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	hash, err := auth.HashPassword("sole-admin-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := "sole-admin-" + uuid.NewString() + "@example.com"
	admin, err := st.CreateUser(ctx, email, "Sole Admin", &hash, store.RoleAdmin, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	isolateEnabledAdmins(t, st, admin.ID)

	_, err = st.UpdateUser(ctx, admin.ID, admin.Email, admin.Name, store.RoleOperator)
	if !errors.Is(err, store.ErrLastAdmin) {
		t.Fatalf("demote sole admin err=%v want ErrLastAdmin", err)
	}

	_, err = st.SetUserDisabled(ctx, admin.ID, true)
	if !errors.Is(err, store.ErrLastAdmin) {
		t.Fatalf("disable sole admin err=%v want ErrLastAdmin", err)
	}

	_, err = st.DeleteUser(ctx, admin.ID)
	if !errors.Is(err, store.ErrLastAdmin) {
		t.Fatalf("delete sole admin err=%v want ErrLastAdmin", err)
	}

	count, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}
	if count != 1 {
		t.Fatalf("CountAdmins = %d want 1", count)
	}
}

func TestLastAdminGuardStoreConcurrentDemote(t *testing.T) {
	testLastAdminConcurrentMutation(t, func(ctx context.Context, st *store.Store, admin *store.User) error {
		_, err := st.UpdateUser(ctx, admin.ID, admin.Email, admin.Name, store.RoleOperator)
		return err
	})
}

func TestLastAdminGuardStoreConcurrentDisable(t *testing.T) {
	testLastAdminConcurrentMutation(t, func(ctx context.Context, st *store.Store, admin *store.User) error {
		_, err := st.SetUserDisabled(ctx, admin.ID, true)
		return err
	})
}

func TestLastAdminGuardStoreConcurrentDelete(t *testing.T) {
	testLastAdminConcurrentMutation(t, func(ctx context.Context, st *store.Store, admin *store.User) error {
		_, err := st.DeleteUser(ctx, admin.ID)
		return err
	})
}

func testLastAdminConcurrentMutation(t *testing.T, mutate func(context.Context, *store.Store, *store.User) error) {
	ctx := context.Background()
	st := openTestStore(t)

	hash, err := auth.HashPassword("concurrent-admin-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	admin1, err := st.CreateUser(ctx, "concurrent-admin-1-"+uuid.NewString()+"@example.com", "Admin One", &hash, store.RoleAdmin, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create admin1: %v", err)
	}
	admin2, err := st.CreateUser(ctx, "concurrent-admin-2-"+uuid.NewString()+"@example.com", "Admin Two", &hash, store.RoleAdmin, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create admin2: %v", err)
	}
	isolateEnabledAdmins(t, st, admin1.ID, admin2.ID)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	admins := []*store.User{admin1, admin2}
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = mutate(ctx, st, admins[idx])
		}(i)
	}
	wg.Wait()

	var lastAdminErrors int
	var successes int
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, store.ErrLastAdmin):
			lastAdminErrors++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successes=%d want 1", successes)
	}
	if lastAdminErrors != 1 {
		t.Fatalf("lastAdminErrors=%d want 1", lastAdminErrors)
	}

	count, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}
	if count < 1 {
		t.Fatalf("CountAdmins = %d want at least 1", count)
	}
}

func isolateEnabledAdmins(t *testing.T, st *store.Store, keepIDs ...uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	keep := make(map[uuid.UUID]struct{}, len(keepIDs))
	for _, id := range keepIDs {
		keep[id] = struct{}{}
	}

	type demotedAdmin struct {
		id   uuid.UUID
		role store.Role
	}
	var demoted []demotedAdmin

	const pageSize = 100
	offset := 0
	for {
		users, total, err := st.ListUsers(ctx, store.ListParams{
			Sort:   "created_at",
			Order:  "asc",
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		for _, u := range users {
			if _, ok := keep[u.ID]; ok || u.Role != store.RoleAdmin || u.DisabledAt != nil {
				continue
			}
			if _, err := st.UpdateUser(ctx, u.ID, u.Email, u.Name, store.RoleOperator); err != nil {
				t.Fatalf("demote leftover admin %s: %v", u.ID, err)
			}
			demoted = append(demoted, demotedAdmin{id: u.ID, role: store.RoleAdmin})
		}
		offset += len(users)
		if len(users) == 0 || offset >= int(total) {
			break
		}
	}

	t.Cleanup(func() {
		for _, d := range demoted {
			existing, err := st.GetUserByID(ctx, d.id)
			if err != nil || existing == nil {
				continue
			}
			_, _ = st.UpdateUser(ctx, existing.ID, existing.Email, existing.Name, d.role)
		}
	})
}

func TestOIDCUserLookupAndLink(t *testing.T) {
	databaseURL := config.TestDatabaseURL(t)

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
