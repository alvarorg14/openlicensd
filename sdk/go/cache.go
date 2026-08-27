package openlicensd

import (
	"context"
	"sync"
	"time"
)

// CachedValidator caches Validate results for a TTL to reduce server round-trips.
type CachedValidator struct {
	client *Client
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	result    ValidationResult
	expiresAt time.Time
}

// NewCachedValidator wraps a client with a TTL cache over Validate.
func NewCachedValidator(client *Client, ttl time.Duration) *CachedValidator {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &CachedValidator{
		client: client,
		ttl:    ttl,
		cache:  make(map[string]cacheEntry),
	}
}

// Validate returns a cached result when available and not expired.
func (v *CachedValidator) Validate(ctx context.Context, key string) (ValidationResult, error) {
	cacheKey := v.client.product + "\x00" + NormalizeKey(key) + "\x00" + v.client.fingerprint

	v.mu.RLock()
	if entry, ok := v.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		v.mu.RUnlock()
		return entry.result, nil
	}
	v.mu.RUnlock()

	result, err := v.client.Validate(ctx, key)
	if err != nil {
		return ValidationResult{}, err
	}

	v.mu.Lock()
	v.cache[cacheKey] = cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(v.ttl),
	}
	v.mu.Unlock()

	return result, nil
}

// Invalidate removes a key from the cache.
func (v *CachedValidator) Invalidate(key string) {
	cacheKey := v.client.product + "\x00" + NormalizeKey(key) + "\x00" + v.client.fingerprint
	v.mu.Lock()
	delete(v.cache, cacheKey)
	v.mu.Unlock()
}

// Clear removes all cached entries.
func (v *CachedValidator) Clear() {
	v.mu.Lock()
	v.cache = make(map[string]cacheEntry)
	v.mu.Unlock()
}
