package store_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTakeRateLimitTokenBurstThenDeny(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	scope := "public"
	key := "burst-" + uuid.NewString()
	burst := 2.0
	refill := 60.0

	for i := 0; i < 2; i++ {
		available, err := st.TakeRateLimitToken(ctx, scope, key, burst, refill)
		if err != nil {
			t.Fatalf("TakeRateLimitToken %d: %v", i, err)
		}
		if available < 1 {
			t.Fatalf("request %d available=%v want >= 1", i, available)
		}
	}

	available, err := st.TakeRateLimitToken(ctx, scope, key, burst, refill)
	if err != nil {
		t.Fatalf("TakeRateLimitToken deny: %v", err)
	}
	if available >= 1 {
		t.Fatalf("expected deny, available=%v", available)
	}
}

func TestTakeRateLimitTokenRefillsAfterWait(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	scope := "public"
	key := "refill-" + uuid.NewString()
	burst := 1.0
	refill := 120.0

	available, err := st.TakeRateLimitToken(ctx, scope, key, burst, refill)
	if err != nil {
		t.Fatalf("first take: %v", err)
	}
	if available < 1 {
		t.Fatalf("available=%v want >= 1", available)
	}

	available, err = st.TakeRateLimitToken(ctx, scope, key, burst, refill)
	if err != nil {
		t.Fatalf("second take: %v", err)
	}
	if available >= 1 {
		t.Fatal("expected second take to be denied")
	}

	time.Sleep(600 * time.Millisecond)

	available, err = st.TakeRateLimitToken(ctx, scope, key, burst, refill)
	if err != nil {
		t.Fatalf("third take: %v", err)
	}
	if available < 1 {
		t.Fatalf("expected refill, available=%v", available)
	}
}

func TestTakeRateLimitTokenScopesAreIsolated(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	key := "scope-" + uuid.NewString()
	burst := 1.0
	refill := 60.0

	available, err := st.TakeRateLimitToken(ctx, "login", key, burst, refill)
	if err != nil {
		t.Fatalf("login take: %v", err)
	}
	if available < 1 {
		t.Fatalf("login available=%v", available)
	}

	available, err = st.TakeRateLimitToken(ctx, "login", key, burst, refill)
	if err != nil {
		t.Fatalf("login deny: %v", err)
	}
	if available >= 1 {
		t.Fatal("expected login scope to deny second request")
	}

	available, err = st.TakeRateLimitToken(ctx, "public", key, burst, refill)
	if err != nil {
		t.Fatalf("public take: %v", err)
	}
	if available < 1 {
		t.Fatalf("public available=%v", available)
	}
}

func TestDeleteIdleRateLimitBucketsRemovesStaleRows(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	scope := "public"
	key := "idle-" + uuid.NewString()
	if _, err := st.TakeRateLimitToken(ctx, scope, key, 2, 60); err != nil {
		t.Fatalf("seed bucket: %v", err)
	}

	removed, err := st.DeleteIdleRateLimitBuckets(ctx, time.Nanosecond)
	if err != nil {
		t.Fatalf("DeleteIdleRateLimitBuckets: %v", err)
	}
	if removed < 1 {
		t.Fatalf("removed=%d want at least 1", removed)
	}
}

func TestTakeRateLimitTokenConcurrentBurst(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	scope := "public"
	key := "concurrent-" + uuid.NewString()
	burst := 5.0
	refill := 60.0

	const workers = 20
	var allowed atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			available, err := st.TakeRateLimitToken(ctx, scope, key, burst, refill)
			if err != nil {
				t.Errorf("TakeRateLimitToken: %v", err)
				return
			}
			if available >= 1 {
				allowed.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := allowed.Load(); got > int64(burst) {
		t.Fatalf("allowed=%d want <= %d", got, int(burst))
	}
	if got := allowed.Load(); got == 0 {
		t.Fatal("expected at least one admission")
	}
}

func TestTakeRateLimitTokenUsesDistinctKeys(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	scope := "public"
	burst := 1.0
	refill := 60.0

	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("distinct-%s-%d", uuid.NewString(), i)
		available, err := st.TakeRateLimitToken(ctx, scope, key, burst, refill)
		if err != nil {
			t.Fatalf("TakeRateLimitToken %d: %v", i, err)
		}
		if available < 1 {
			t.Fatalf("request %d available=%v", i, available)
		}
	}
}
