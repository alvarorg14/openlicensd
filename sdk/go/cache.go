package openlicensd

import (
	"context"
	"sync"
	"time"
)

// CachedValidator caches Validate results for a TTL to reduce server round-trips.
//
// Any ValidationResult returned without error is cached, including Valid=false.
// Transport errors are not cached. The cache map has no size limit; expired
// entries are skipped on read but not pruned automatically. Use Invalidate or
// Clear to remove entries when many distinct keys are validated.
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

func (v *CachedValidator) entryKey(key, product string) string {
	if product == "" {
		product = v.client.product
	}
	return product + "\x00" + NormalizeKey(key) + "\x00" + v.client.fingerprint
}

// Validate returns a cached result when available and not expired. Invalid
// licenses (Valid=false) are cached for the TTL when Validate returns a nil
// error. Transport errors are returned immediately and are not stored.
func (v *CachedValidator) Validate(ctx context.Context, key string) (ValidationResult, error) {
	return v.ValidateProduct(ctx, key, v.client.product)
}

// ValidateProduct returns a cached result for key scoped to product. When product
// is empty, the client's configured product is used.
func (v *CachedValidator) ValidateProduct(ctx context.Context, key, product string) (ValidationResult, error) {
	cacheKey := v.entryKey(key, product)

	v.mu.RLock()
	if entry, ok := v.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		v.mu.RUnlock()
		return entry.result, nil
	}
	v.mu.RUnlock()

	result, err := v.client.ValidateProduct(ctx, key, product)
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

// Invalidate removes a key from the cache for the client's configured product.
func (v *CachedValidator) Invalidate(key string) {
	v.InvalidateProduct(key, v.client.product)
}

// InvalidateProduct removes a key from the cache for the given product. When
// product is empty, the client's configured product is used.
func (v *CachedValidator) InvalidateProduct(key, product string) {
	cacheKey := v.entryKey(key, product)
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
