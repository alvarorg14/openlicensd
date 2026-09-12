package maintenance_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/maintenance"
)

type fakeAuditEventStore struct {
	mu      sync.Mutex
	calls   int
	removed int64
	err     error
	lastCutoff time.Time
}

func (f *fakeAuditEventStore) DeleteAuditEventsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastCutoff = cutoff
	if f.err != nil {
		return 0, f.err
	}
	return f.removed, nil
}

func (f *fakeAuditEventStore) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestAuditEventPrunerRunsImmediatelyAndOnTicker(t *testing.T) {
	store := &fakeAuditEventStore{removed: 2}
	pruner := maintenance.NewAuditEventPruner(store, 30, 20*time.Millisecond, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		pruner.Run(ctx)
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
		t.Fatal("pruner did not stop after context cancellation")
	}

	if store.callCount() < 2 {
		t.Fatalf("calls=%d want at least 2 (immediate + ticker)", store.callCount())
	}
}

func TestAuditEventPrunerContinuesAfterError(t *testing.T) {
	store := &fakeAuditEventStore{err: errors.New("db unavailable")}
	pruner := maintenance.NewAuditEventPruner(store, 30, 10*time.Millisecond, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		pruner.Run(ctx)
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
		t.Fatal("pruner did not stop after context cancellation")
	}

	if store.callCount() < 2 {
		t.Fatalf("calls=%d want at least 2 after errors", store.callCount())
	}
}

func TestAuditEventPrunerRunOnce(t *testing.T) {
	store := &fakeAuditEventStore{removed: 3}
	pruner := maintenance.NewAuditEventPruner(store, 30, time.Minute, testLogger())

	before := time.Now().UTC()
	removed, err := pruner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed=%d want 3", removed)
	}
	if store.callCount() != 1 {
		t.Fatalf("calls=%d want 1", store.callCount())
	}

	store.mu.Lock()
	cutoff := store.lastCutoff
	store.mu.Unlock()

	expectedCutoff := before.Add(-30 * 24 * time.Hour)
	if cutoff.Before(expectedCutoff.Add(-2*time.Second)) || cutoff.After(expectedCutoff.Add(2*time.Second)) {
		t.Fatalf("cutoff=%s want near %s", cutoff, expectedCutoff)
	}
}
