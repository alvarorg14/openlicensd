package store_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/openlicensd/openlicensd/server/internal/store"
)

func TestMigrateConcurrent(t *testing.T) {
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

	const workers = 4
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := store.New(ctx, databaseURL)
			if err != nil {
				errCh <- err
				return
			}
			defer s.Close()
			errCh <- s.Migrate(ctx)
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
}
