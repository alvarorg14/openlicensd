package maintenance_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/maintenance"
)

type fakeSessionStore struct {
	mu       sync.Mutex
	calls    int
	removed  int64
	err      error
}

func (f *fakeSessionStore) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return 0, f.err
	}
	return f.removed, nil
}

func (f *fakeSessionStore) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestSessionCleanerRunsImmediatelyAndOnTicker(t *testing.T) {
	store := &fakeSessionStore{removed: 2}
	cleaner := maintenance.NewSessionCleaner(store, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		cleaner.Run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if store.callCount() >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleaner did not stop after context cancellation")
	}

	if store.callCount() < 2 {
		t.Fatalf("calls=%d want at least 2 (immediate + ticker)", store.callCount())
	}
}

func TestSessionCleanerContinuesAfterError(t *testing.T) {
	store := &fakeSessionStore{err: errors.New("db unavailable")}
	cleaner := maintenance.NewSessionCleaner(store, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		cleaner.Run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if store.callCount() >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleaner did not stop after context cancellation")
	}

	if store.callCount() < 2 {
		t.Fatalf("calls=%d want at least 2 after errors", store.callCount())
	}
}

func TestSessionCleanerRunOnce(t *testing.T) {
	store := &fakeSessionStore{removed: 3}
	cleaner := maintenance.NewSessionCleaner(store, time.Minute)

	removed, err := cleaner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed=%d want 3", removed)
	}
	if store.callCount() != 1 {
		t.Fatalf("calls=%d want 1", store.callCount())
	}
}
