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

// WithProduct sets the product code used for revalidation. When empty, the
// guard uses the client's configured product.
func WithProduct(product string) GuardOption {
	return func(g *Guard) {
		g.product = product
	}
}

// Guard periodically revalidates a license key and exposes the latest result.
// After a successful start, it tolerates transient network failures within an
// offline grace window configured by WithOfflineGrace.
type Guard struct {
	client  *Client
	key     string
	product string

	interval     time.Duration
	offlineGrace time.Duration

	mu              sync.RWMutex
	last            ValidationResult
	lastValidatedAt time.Time
	valid           bool
	lastErr         error

	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func (g *Guard) validationProduct() string {
	if g.product != "" {
		return g.product
	}
	return g.client.product
}

// NewGuard starts background revalidation for key. Call Stop to release resources.
//
// The first ValidateProduct call runs synchronously. If it returns a non-nil error
// (for example when the server is unreachable), NewGuard returns that error
// and does not start the background loop. Offline grace applies only to later
// transport failures after a successful start. An invalid license (Valid=false
// with a nil error) still constructs the guard.
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

	result, err := client.ValidateProduct(ctx, key, g.validationProduct())
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
	result, err := g.client.ValidateProduct(ctx, g.key, g.validationProduct())

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
// It is safe to call Stop more than once.
func (g *Guard) Stop() {
	g.stopOnce.Do(func() {
		close(g.stop)
		<-g.done
	})
}
