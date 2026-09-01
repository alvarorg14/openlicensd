package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

func TestAPITokenCRUD(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	_, hash, prefix, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate api token: %v", err)
	}

	name := "ci-token-" + uuid.NewString()
	tok, err := st.CreateAPIToken(ctx, name, hash, prefix, store.RoleOperator, nil, nil)
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}
	if tok.TokenPrefix != prefix {
		t.Fatalf("token_prefix=%q want %q", tok.TokenPrefix, prefix)
	}

	dup, err := st.CreateAPIToken(ctx, name, hash+"x", prefix+"x", store.RoleViewer, nil, nil)
	if err == nil {
		t.Fatalf("expected duplicate name conflict, got token %#v", dup)
	}
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	byHash, err := st.GetAPITokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("get by hash: %v", err)
	}
	if byHash == nil || byHash.ID != tok.ID {
		t.Fatalf("expected token by hash")
	}

	byID, err := st.GetAPITokenByID(ctx, tok.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID == nil || byID.Name != name {
		t.Fatalf("expected token by id")
	}

	if err := st.TouchAPIToken(ctx, tok.ID); err != nil {
		t.Fatalf("touch api token: %v", err)
	}
	touched, err := st.GetAPITokenByID(ctx, tok.ID)
	if err != nil {
		t.Fatalf("get after touch: %v", err)
	}
	if touched.LastUsedAt == nil {
		t.Fatalf("expected last_used_at to be set")
	}

	revoked, err := st.RevokeAPIToken(ctx, tok.ID)
	if err != nil {
		t.Fatalf("revoke api token: %v", err)
	}
	if revoked == nil || revoked.RevokedAt == nil {
		t.Fatalf("expected revoked token")
	}

	afterRevoke, err := st.GetAPITokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("get revoked by hash: %v", err)
	}
	if afterRevoke != nil {
		t.Fatalf("expected revoked token to be hidden from hash lookup")
	}

	// Recreate for delete test.
	raw2, hash2, prefix2, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate api token: %v", err)
	}
	_ = raw2
	tok2, err := st.CreateAPIToken(ctx, "delete-me-"+uuid.NewString(), hash2, prefix2, store.RoleAdmin, nil, nil)
	if err != nil {
		t.Fatalf("create api token for delete: %v", err)
	}

	deleted, err := st.DeleteAPIToken(ctx, tok2.ID)
	if err != nil {
		t.Fatalf("delete api token: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete to succeed")
	}

	missing, err := st.DeleteAPIToken(ctx, uuid.New())
	if err != nil {
		t.Fatalf("delete missing api token: %v", err)
	}
	if missing {
		t.Fatalf("expected delete missing to return false")
	}
}

func TestAPITokenExpiredHiddenFromLookup(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	_, hash, prefix, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate api token: %v", err)
	}

	expired := time.Now().Add(-time.Hour)
	_, err = st.CreateAPIToken(ctx, "expired-"+uuid.NewString(), hash, prefix, store.RoleViewer, nil, &expired)
	if err != nil {
		t.Fatalf("create expired api token: %v", err)
	}

	tok, err := st.GetAPITokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("get expired by hash: %v", err)
	}
	if tok != nil {
		t.Fatalf("expected expired token to be hidden from hash lookup")
	}
}

func TestListAPITokens(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	name := "list-token-" + uuid.NewString()
	_, hash, prefix, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatalf("generate api token: %v", err)
	}
	if _, err := st.CreateAPIToken(ctx, name, hash, prefix, store.RoleAdmin, nil, nil); err != nil {
		t.Fatalf("create api token: %v", err)
	}

	tokens, total, err := st.ListAPITokens(ctx, store.ListParams{
		Search: name,
		Sort:   "name",
		Order:  "asc",
		Limit:  25,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list api tokens: %v", err)
	}
	if total < 1 || len(tokens) < 1 {
		t.Fatalf("expected at least one token, total=%d len=%d", total, len(tokens))
	}
}
