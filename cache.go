package gotrans

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// CacheStats provides statistics about cache performance.
type CacheStats struct {
	Hits        int64
	Misses      int64
	Sets        int64
	Deletes     int64
	LastCleared time.Time
}

// TranslationCache is the interface for pluggable cache backends.
// Implementations can be in-memory, Redis, Memcached, or any custom store.
type TranslationCache interface {
	// Get retrieves cached translations. Returns the slice and true on a hit.
	Get(key string) ([]Translation, bool)

	// Set stores translations under the given key. A zero TTL means no expiration.
	Set(key string, value []Translation, ttl time.Duration)

	// Delete removes one or more entries by key.
	Delete(keys ...string)

	// Clear removes all entries from the cache.
	Clear()

	// Stats returns cache statistics (hits, misses, etc).
	Stats() CacheStats

	// ResetStats resets cache statistics to zero.
	ResetStats()
}

// CacheOptions configures the caching behaviour of a cached repository.
type CacheOptions struct {
	// TTL defines how long a cached entry remains valid.
	// Zero means entries never expire.
	TTL time.Duration

	// BatchSize defines the maximum number of IDs to fetch in a single query.
	// Default is 1000 if not set or set to 0.
	BatchSize int

	// DefaultContextTimeout specifies default timeout for cache operations.
	// Zero means no timeout.
	DefaultContextTimeout time.Duration
}

// -----------------------------------------------------------------------------
// In-memory cache implementation
// -----------------------------------------------------------------------------

type cacheEntry struct {
	value     []Translation
	expiresAt time.Time // zero means no expiration
}

func (e cacheEntry) isExpired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// InMemoryCache is a thread-safe in-memory implementation of TranslationCache.
// It is the default backend used by NewCachedRepositoryInMemory.
type InMemoryCache struct {
	mu      sync.RWMutex
	items   map[string]cacheEntry
	stats   CacheStats
	statsMu sync.RWMutex
}

// NewInMemoryCache returns a ready-to-use in-memory cache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{items: make(map[string]cacheEntry)}
}

func (c *InMemoryCache) Get(key string) ([]Translation, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		c.statsMu.Lock()
		c.stats.Misses++
		c.statsMu.Unlock()
		return nil, false
	}

	if entry.isExpired() {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		c.statsMu.Lock()
		c.stats.Misses++
		c.statsMu.Unlock()
		return nil, false
	}

	c.statsMu.Lock()
	c.stats.Hits++
	c.statsMu.Unlock()
	return entry.value, true
}

func (c *InMemoryCache) Set(key string, value []Translation, ttl time.Duration) {
	entry := cacheEntry{value: value}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.items[key] = entry
	c.mu.Unlock()
	c.statsMu.Lock()
	c.stats.Sets++
	c.statsMu.Unlock()
}

func (c *InMemoryCache) Delete(keys ...string) {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.items, k)
	}
	c.mu.Unlock()
	c.statsMu.Lock()
	c.stats.Deletes += int64(len(keys))
	c.statsMu.Unlock()
}

func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]cacheEntry)
	c.mu.Unlock()
	c.statsMu.Lock()
	c.stats.LastCleared = time.Now()
	c.statsMu.Unlock()
}

// Stats returns cache statistics.
func (c *InMemoryCache) Stats() CacheStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats
}

// ResetStats resets cache statistics to zero.
func (c *InMemoryCache) ResetStats() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats = CacheStats{}
}

// -----------------------------------------------------------------------------
// Cached repository — transparent decorator over TranslationRepository
// -----------------------------------------------------------------------------

// cachedRepository wraps any TranslationRepository and adds a caching layer.
// Cache keys are per (locale, entity, entityID), so partial hits are supported:
// only IDs missing from cache are fetched from the underlying store.
//
// An entity index tracks all cache keys per (entity, entityID) so that a
// "delete all locales" operation (LocaleNone) can invalidate every locale entry
// without scanning the whole cache.
type cachedRepository struct {
	repo  TranslationRepository
	cache TranslationCache
	opts  CacheOptions

	idxMu       sync.RWMutex
	entityIndex map[string]map[string]struct{} // "entity:id" → set of cache keys
}

// NewCachedRepository wraps repo with the provided cache backend.
// Use this when you want to supply your own cache implementation (e.g. Redis).
//
//	cache := gotrans.NewInMemoryCache()
//	cachedRepo := gotrans.NewCachedRepository(repo, cache, gotrans.CacheOptions{TTL: 5 * time.Minute})
//	translator := gotrans.NewTranslator[Product](cachedRepo)
func NewCachedRepository(repo TranslationRepository, cache TranslationCache, opts CacheOptions) TranslationRepository {
	return &cachedRepository{
		repo:        repo,
		cache:       cache,
		opts:        opts,
		entityIndex: make(map[string]map[string]struct{}),
	}
}

// NewCachedRepositoryInMemory wraps repo with the built-in in-memory cache.
// This is the simplest way to add caching with no external dependencies.
//
//	cachedRepo := gotrans.NewCachedRepositoryInMemory(repo, gotrans.CacheOptions{TTL: 5 * time.Minute})
//	translator := gotrans.NewTranslator[Product](cachedRepo)
func NewCachedRepositoryInMemory(repo TranslationRepository, opts CacheOptions) TranslationRepository {
	return NewCachedRepository(repo, NewInMemoryCache(), opts)
}

// GetTranslations checks the cache per entity ID and fetches only the missing
// IDs from the underlying repository (cache-aside pattern). Uses batch processing
// with configurable batch size (default 1000).
func (c *cachedRepository) GetTranslations(
	ctx context.Context,
	locale Locale,
	entity string,
	entityIDs []int,
) ([]Translation, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	var result []Translation
	var missedIDs []int

	for _, id := range entityIDs {
		key := translationCacheKey(locale, entity, id)
		if cached, ok := c.cache.Get(key); ok {
			result = append(result, cached...)
		} else {
			missedIDs = append(missedIDs, id)
		}
	}

	if len(missedIDs) == 0 {
		return result, nil
	}

	// Determine batch size
	batchSize := c.opts.BatchSize
	if batchSize <= 0 {
		batchSize = 1000 // default batch size
	}

	var fetched []Translation

	// Process in batches
	for start := 0; start < len(missedIDs); start += batchSize {
		end := start + batchSize
		if end > len(missedIDs) {
			end = len(missedIDs)
		}

		batch, err := c.repo.GetTranslations(ctx, locale, entity, missedIDs[start:end])
		if err != nil {
			return nil, err
		}
		fetched = append(fetched, batch...)
	}

	// Group by entityID so we can cache each entity separately.
	byID := make(map[int][]Translation, len(missedIDs))
	for _, tr := range fetched {
		byID[tr.EntityID] = append(byID[tr.EntityID], tr)
	}

	// Store in cache — ОДНА операция под lock!
	c.idxMu.Lock()
	for _, id := range missedIDs {
		key := translationCacheKey(locale, entity, id)
		translations := byID[id]
		if translations == nil {
			translations = []Translation{}
		}
		c.cache.Set(key, translations, c.opts.TTL)
		// Встроена логика trackKey для атомарности
		eKey := entityIndexKey(entity, id)
		if c.entityIndex[eKey] == nil {
			c.entityIndex[eKey] = make(map[string]struct{})
		}
		c.entityIndex[eKey][key] = struct{}{}
	}
	c.idxMu.Unlock()

	return append(result, fetched...), nil
}

func (c *cachedRepository) MassDelete(
	ctx context.Context,
	locale Locale,
	entity string,
	entityIDs []int,
	fields []string,
) error {
	if err := c.repo.MassDelete(ctx, locale, entity, entityIDs, fields); err != nil {
		return err
	}
	if locale == LocaleNone {
		// LocaleNone means all locales — use the entity index to invalidate every
		// locale variant without knowing which locales were cached.
		c.invalidateAllLocales(entity, entityIDs)
	} else {
		keys := make([]string, 0, len(entityIDs))
		for _, id := range entityIDs {
			keys = append(keys, translationCacheKey(locale, entity, id))
		}
		c.cache.Delete(keys...)
		c.untrackKeys(entity, entityIDs, keys)
	}
	return nil
}

func (c *cachedRepository) MassCreateOrUpdate(
	ctx context.Context,
	locale Locale,
	translations []Translation,
) error {
	if err := c.repo.MassCreateOrUpdate(ctx, locale, translations); err != nil {
		return err
	}
	c.invalidateByTranslations(translations)
	return nil
}

// invalidateByTranslations removes cache entries for the affected translations.
// The idxMu lock is held for the entire operation (cache.Delete + index update)
// to prevent a race where another goroutine could re-add a key between the two steps.
func (c *cachedRepository) invalidateByTranslations(translations []Translation) {
	seen := make(map[string]struct{}, len(translations))
	keys := make([]string, 0, len(translations))
	for _, tr := range translations {
		k := translationCacheKey(tr.Locale, tr.Entity, tr.EntityID)
		if _, dup := seen[k]; !dup {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}

	c.idxMu.Lock()
	defer c.idxMu.Unlock()
	c.cache.Delete(keys...)
	for _, tr := range translations {
		eKey := entityIndexKey(tr.Entity, tr.EntityID)
		if keyset, ok := c.entityIndex[eKey]; ok {
			delete(keyset, translationCacheKey(tr.Locale, tr.Entity, tr.EntityID))
		}
	}
}

// invalidateAllLocales removes all cached entries for the given entity IDs
// across every locale, using the entity index.
func (c *cachedRepository) invalidateAllLocales(entity string, entityIDs []int) {
	c.idxMu.Lock()
	defer c.idxMu.Unlock()
	for _, id := range entityIDs {
		eKey := entityIndexKey(entity, id)
		if keyset, ok := c.entityIndex[eKey]; ok {
			keys := make([]string, 0, len(keyset))
			for k := range keyset {
				keys = append(keys, k)
			}
			c.cache.Delete(keys...)
			delete(c.entityIndex, eKey)
		}
	}
}

// untrackKeys removes keys from the entity index after explicit deletion.
func (c *cachedRepository) untrackKeys(entity string, entityIDs []int, keys []string) {
	c.idxMu.Lock()
	for i, id := range entityIDs {
		eKey := entityIndexKey(entity, id)
		if keyset, ok := c.entityIndex[eKey]; ok {
			delete(keyset, keys[i])
		}
	}
	c.idxMu.Unlock()
}

// translationCacheKey builds the per-entity-locale cache key.
// Uses string concatenation instead of fmt.Sprintf for better hot-path performance.
func translationCacheKey(locale Locale, entity string, entityID int) string {
	if entity == "" {
		entity = "unknown"
	}
	return locale.String() + ":" + entity + ":" + strconv.Itoa(entityID)
}

// entityIndexKey builds the entity-level index key used for cross-locale invalidation.
func entityIndexKey(entity string, entityID int) string {
	return entity + ":" + strconv.Itoa(entityID)
}
