package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDeleteProductForeignKey(t *testing.T) {
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

	product, err := st.CreateProduct(ctx, "Delete FK Test", uuid.NewString(), nil)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	_, err = st.CreatePolicy(ctx, product.ID, "Policy", nil, nil, store.ExpirationOnCreation, 0, nil)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	_, err = st.DeleteProduct(ctx, product.ID)
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %T %v", err, err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		t.Fatalf("expected mapped conflict, not raw pg error code=%s", pgErr.Code)
	}
}
