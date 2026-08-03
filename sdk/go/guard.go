package openlicensd

import (
	"context"
	"sync"
	"time"
)

// GuardOption configures a Guard.
type GuardOption func(*Guard)

// WithInterval sets how often the guard revalidates the license.
func WithInterval(d time.Duration) GuardOption {
	return func(g *Guard) {
		g.interval = d
	}
}

// WithOfflineGrace sets how long the guard remains valid after the last
// successful validation when the server becomes unreachable.
func WithOfflineGrace(d time.Duration) GuardOption {
	return func(g *Guard) {
		g.offlineGrace = d
	}
}

// Guard periodically revalidates a license key and exposes the latest result.
// It tolerates transient network failures within an offline grace window.
type Guard struct {
	client *Client
	key    string

	interval     time.Duration
	offlineGrace time.Duration

	mu              sync.RWMutex
	last            ValidationResult
	lastValidatedAt time.Time
	valid           bool
	lastErr         error

	stop chan struct{}
	done chan struct{}
}

// NewGuard starts background revalidation for key. Call Stop to release resources.
func NewGuard(ctx context.Context, client *Client, key string, opts ...GuardOption) (*Guard, error) {
	g := &Guard{
		client:       client,
		key:          key,
		interval:     time.Hour,
		offlineGrace: 24 * time.Hour,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}

	for _, opt := range opts {
		opt(g)
	}

	result, err := client.Validate(ctx, key)
	g.mu.Lock()
	g.last = result
	g.lastValidatedAt = time.Now()
	g.valid = result.Valid
	g.lastErr = err
	g.mu.Unlock()

	if err != nil {
		return nil, err
	}

	go g.run(ctx)
	return g, nil
}

func (g *Guard) run(ctx context.Context) {
	defer close(g.done)

	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stop:
			return
		case <-ticker.C:
			g.revalidate(ctx)
		}
	}
}

func (g *Guard) revalidate(ctx context.Context) {
	result, err := g.client.Validate(ctx, g.key)

	g.mu.Lock()
	defer g.mu.Unlock()

	if err == nil {
		g.last = result
		g.lastValidatedAt = time.Now()
		g.valid = result.Valid
		g.lastErr = nil
		return
	}

	g.lastErr = err
	if g.valid && time.Since(g.lastValidatedAt) <= g.offlineGrace {
		return
	}
	g.valid = false
}

// Valid reports whether the license is currently considered valid.
func (g *Guard) Valid() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.valid
}

// Last returns the most recent validation result.
func (g *Guard) Last() ValidationResult {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.last
}

// LastError returns the most recent validation error, if any.
func (g *Guard) LastError() error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastErr
}

// Stop ends background revalidation and waits for the goroutine to exit.
func (g *Guard) Stop() {
	close(g.stop)
	<-g.done
}
