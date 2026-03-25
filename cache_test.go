package gotrans
import (
"context"
"testing"
"time"
"github.com/stretchr/testify/require"
)
type countingRepo struct {
	mockRepo
	getCalls int
}
func (r *countingRepo) GetTranslations(ctx context.Context, locale Locale, entity string, ids []int) ([]Translation, error) {
	r.getCalls++
	return r.mockRepo.GetTranslations(ctx, locale, entity, ids)
}
func TestInMemoryCache_SetGet(t *testing.T) {
	c := NewInMemoryCache()
	key := "en:product:1"
	value := []Translation{{ID: 1, Entity: "product", EntityID: 1, Field: "title", Locale: LocaleEN, Value: "Apple"}}
	_, ok := c.Get(key)
	require.False(t, ok)
	c.Set(key, value, 0)
	got, ok := c.Get(key)
	require.True(t, ok)
	require.Equal(t, value, got)
}
func TestInMemoryCache_TTLExpiry(t *testing.T) {
	c := NewInMemoryCache()
	key := "en:product:1"
	c.Set(key, []Translation{}, 20*time.Millisecond)
	_, ok := c.Get(key)
	require.True(t, ok)
	time.Sleep(30 * time.Millisecond)
	_, ok = c.Get(key)
	require.False(t, ok, "expected miss after TTL")
}
func TestInMemoryCache_Delete(t *testing.T) {
	c := NewInMemoryCache()
	c.Set("k1", []Translation{}, 0)
	c.Set("k2", []Translation{}, 0)
	c.Delete("k1")
	_, ok := c.Get("k1")
	require.False(t, ok)
	_, ok = c.Get("k2")
	require.True(t, ok)
}
func TestInMemoryCache_Clear(t *testing.T) {
	c := NewInMemoryCache()
	c.Set("k1", []Translation{}, 0)
	c.Set("k2", []Translation{}, 0)
	c.Clear()
	_, ok := c.Get("k1")
	require.False(t, ok)
	_, ok = c.Get("k2")
	require.False(t, ok)
}
func TestCachedRepository_CacheHit(t *testing.T) {
	base := &countingRepo{mockRepo: mockRepo{
		translations: []Translation{{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Hello"}},
	}}
	repo := NewCachedRepositoryInMemory(base, CacheOptions{TTL: time.Minute})
	ctx := context.Background()
	res, err := repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, 1, base.getCalls)
	res, err = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, 1, base.getCalls, "second call must be served from cache")
}
func TestCachedRepository_PartialCacheHit(t *testing.T) {
	base := &countingRepo{mockRepo: mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "One"},
			{ID: 2, Entity: "parameter", EntityID: 2, Field: "name", Locale: LocaleEN, Value: "Two"},
		},
	}}
	repo := NewCachedRepositoryInMemory(base, CacheOptions{TTL: time.Minute})
	ctx := context.Background()
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.Equal(t, 1, base.getCalls)
	res, err := repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1, 2})
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, 2, base.getCalls, "only ID 2 should trigger a DB call")
}
func TestCachedRepository_InvalidationOnUpdate(t *testing.T) {
	base := &countingRepo{mockRepo: mockRepo{
		translations: []Translation{{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Old"}},
	}}
	repo := NewCachedRepositoryInMemory(base, CacheOptions{TTL: time.Minute})
	ctx := context.Background()
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.Equal(t, 1, base.getCalls)
	updated := []Translation{{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "New"}}
	base.mockRepo.translations = updated
	_ = repo.MassCreateOrUpdate(ctx, LocaleEN, updated)
	res, err := repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.NoError(t, err)
	require.Equal(t, 2, base.getCalls, "DB must be called after cache invalidation")
	require.Equal(t, "New", res[0].Value)
}
func TestCachedRepository_InvalidationOnDelete(t *testing.T) {
	base := &countingRepo{mockRepo: mockRepo{
		translations: []Translation{{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Hello"}},
	}}
	repo := NewCachedRepositoryInMemory(base, CacheOptions{TTL: time.Minute})
	ctx := context.Background()
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.Equal(t, 1, base.getCalls)
	base.mockRepo.translations = nil
	_ = repo.MassDelete(ctx, LocaleEN, "parameter", []int{1}, nil)
	res, err := repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	require.NoError(t, err)
	require.Equal(t, 2, base.getCalls)
	require.Empty(t, res)
}
func TestCachedRepository_InvalidationAllLocales(t *testing.T) {
	base := &countingRepo{mockRepo: mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "EN"},
			{ID: 2, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleFR, Value: "FR"},
		},
	}}
	repo := NewCachedRepositoryInMemory(base, CacheOptions{TTL: time.Minute})
	ctx := context.Background()
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	_, _ = repo.GetTranslations(ctx, LocaleFR, "parameter", []int{1})
	require.Equal(t, 2, base.getCalls)
	base.mockRepo.translations = nil
	_ = repo.MassDelete(ctx, LocaleNone, "parameter", []int{1}, nil)
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{1})
	_, _ = repo.GetTranslations(ctx, LocaleFR, "parameter", []int{1})
	require.Equal(t, 4, base.getCalls, "both locale entries must be evicted")
}
func TestCachedRepository_EmptyResultCached(t *testing.T) {
	base := &countingRepo{mockRepo: mockRepo{}}
	repo := NewCachedRepositoryInMemory(base, CacheOptions{TTL: time.Minute})
	ctx := context.Background()
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{99})
	require.Equal(t, 1, base.getCalls)
	_, _ = repo.GetTranslations(ctx, LocaleEN, "parameter", []int{99})
	require.Equal(t, 1, base.getCalls, "empty result must be cached")
}
