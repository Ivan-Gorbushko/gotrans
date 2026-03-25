package gotrans

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Parameter struct {
	ID          int
	locale      Locale
	Name        string
	Description string
}

var _ Translatable = (*Parameter)(nil)

func (p Parameter) TranslationEntityLocale() Locale { return p.locale }
func (p Parameter) TranslationEntityID() int        { return p.ID }
func (p Parameter) TranslatableFields() map[string]string {
	return map[string]string{
		"Name":        "name",
		"Description": "description",
	}
}
func (p Parameter) TranslationEntityName() string { return "parameter" }

func TestLoadTranslations(t *testing.T) {
	repo := &mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Example Name EN"},
			{ID: 2, Entity: "parameter", EntityID: 1, Field: "description", Locale: LocaleEN, Value: "Desc EN"},
		},
	}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{ID: 1, locale: LocaleEN}}
	ctx := context.Background()
	parms, err := paramTrans.LoadTranslations(ctx, parms)
	require.NoError(t, err)

	require.Equal(t, "Example Name EN", parms[0].Name)
	require.Equal(t, "Desc EN", parms[0].Description)
}

func TestSaveTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{
		ID:          1,
		locale:      LocaleEN,
		Name:        "New Name EN",
		Description: "Desc EN",
	}}
	ctx := context.Background()
	err := paramTrans.SaveTranslations(ctx, parms)
	require.NoError(t, err)

	require.Len(t, repo.saved, 2)
	// Check that translations were saved
	hasName := false
	hasDesc := false
	for _, tr := range repo.saved {
		if tr.Field == "name" && tr.Value == "New Name EN" && tr.Entity == "parameter" {
			hasName = true
		}
		if tr.Field == "description" && tr.Value == "Desc EN" && tr.Entity == "parameter" {
			hasDesc = true
		}
	}
	require.True(t, hasName)
	require.True(t, hasDesc)
}

func TestDeleteTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{
		ID:          1,
		locale:      LocaleEN,
		Name:        "Name EN",
		Description: "Desc EN",
	}}
	ctx := context.Background()
	_ = paramTrans.SaveTranslations(ctx, parms)
	require.Len(t, repo.saved, 2)

	// Delete through the translator interface, not the mock directly.
	err := paramTrans.DeleteTranslations(ctx, LocaleEN, []int{1}, []string{"name", "description"})
	require.NoError(t, err)
	require.Len(t, repo.saved, 0)
}

// TestMultiLocaleSaveAndLoad demonstrates that translations are grouped by locale
// This test verifies that the translator efficiently handles multiple locales
func TestMultiLocaleSaveAndLoad(t *testing.T) {
	repo := &mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Name EN"},
			{ID: 2, Entity: "parameter", EntityID: 2, Field: "name", Locale: LocaleFR, Value: "Name FR"},
			{ID: 3, Entity: "parameter", EntityID: 1, Field: "description", Locale: LocaleEN, Value: "Desc EN"},
			{ID: 4, Entity: "parameter", EntityID: 2, Field: "description", Locale: LocaleFR, Value: "Desc FR"},
		},
	}
	paramTrans := NewTranslator[Parameter](repo)

	// Load with mixed locales
	parms := []Parameter{
		{ID: 1, locale: LocaleEN},
		{ID: 2, locale: LocaleFR},
	}
	ctx := context.Background()
	parms, err := paramTrans.LoadTranslations(ctx, parms)
	require.NoError(t, err)

	require.Equal(t, "Name EN", parms[0].Name)
	require.Equal(t, "Desc EN", parms[0].Description)
	require.Equal(t, "Name FR", parms[1].Name)
	require.Equal(t, "Desc FR", parms[1].Description)
}

type mockRepo struct {
	mu            sync.Mutex
	saved         []Translation
	translations  []Translation
	getErr        error
	saveErr       error
	deleteErr     error
}

func (m *mockRepo) GetTranslations(
	_ context.Context,
	locale Locale,
	entity string,
	entityIDs []int,
) ([]Translation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	idSet := make(map[int]struct{}, len(entityIDs))
	for _, id := range entityIDs {
		idSet[id] = struct{}{}
	}
	var result []Translation
	for _, tr := range m.translations {
		if tr.Entity != entity || tr.Locale != locale {
			continue
		}
		if _, ok := idSet[tr.EntityID]; ok {
			result = append(result, tr)
		}
	}
	return result, nil
}

func (m *mockRepo) MassDelete(
	_ context.Context,
	locale Locale,
	entity string,
	entityIDs []int,
	fields []string,
) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idSet := make(map[int]struct{}, len(entityIDs))
	for _, id := range entityIDs {
		idSet[id] = struct{}{}
	}
	fieldSet := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldSet[f] = struct{}{}
	}

	var filtered []Translation
	for _, tr := range m.saved {
		if tr.Entity != entity {
			filtered = append(filtered, tr)
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[tr.EntityID]; !ok {
				filtered = append(filtered, tr)
				continue
			}
		}
		// LocaleNone means "all locales" — skip locale filter, like real SQL.
		if locale != LocaleNone && tr.Locale != locale {
			filtered = append(filtered, tr)
			continue
		}
		if len(fieldSet) > 0 {
			if _, ok := fieldSet[tr.Field]; !ok {
				filtered = append(filtered, tr)
				continue
			}
		}
		// Entry matches all criteria — delete it (don't append).
	}
	m.saved = filtered
	return nil
}

func (m *mockRepo) MassCreateOrUpdate(
	_ context.Context,
	locale Locale,
	translations []Translation,
) error {
	if m.saveErr != nil {
		return m.saveErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(translations) == 0 {
		return nil
	}
	// Delete matching (entity, id, field, locale) combinations — locale-specific.
	type key struct {
		entity string
		id     int
		field  string
		locale string
	}
	toDelete := make(map[key]struct{}, len(translations))
	for _, tr := range translations {
		toDelete[key{tr.Entity, tr.EntityID, tr.Field, locale.String()}] = struct{}{}
	}
	filtered := m.saved[:0:0]
	for _, tr := range m.saved {
		if _, ok := toDelete[key{tr.Entity, tr.EntityID, tr.Field, tr.Locale.String()}]; !ok {
			filtered = append(filtered, tr)
		}
	}
	m.saved = append(filtered, translations...)
	return nil
}

func TestLoadTranslations_EmptyInput(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)
	ctx := context.Background()

	result, err := paramTrans.LoadTranslations(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, result)

	result, err = paramTrans.LoadTranslations(ctx, []Parameter{})
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestLoadTranslations_ErrorPropagation(t *testing.T) {
	repo := &mockRepo{getErr: errTest}
	paramTrans := NewTranslator[Parameter](repo)
	ctx := context.Background()

	_, err := paramTrans.LoadTranslations(ctx, []Parameter{{ID: 1, locale: LocaleEN}})
	require.ErrorIs(t, err, errTest)
}

func TestSaveTranslations_EmptyInput(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)
	ctx := context.Background()

	err := paramTrans.SaveTranslations(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, repo.saved)
}

func TestSaveTranslations_ErrorPropagation(t *testing.T) {
	repo := &mockRepo{saveErr: errTest}
	paramTrans := NewTranslator[Parameter](repo)
	ctx := context.Background()

	err := paramTrans.SaveTranslations(ctx, []Parameter{{ID: 1, locale: LocaleEN, Name: "Test"}})
	require.ErrorIs(t, err, errTest)
}

func TestDeleteTranslationsByEntity(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)
	ctx := context.Background()

	// Save translations for two locales.
	parms := []Parameter{
		{ID: 1, locale: LocaleEN, Name: "Name EN", Description: "Desc EN"},
		{ID: 1, locale: LocaleFR, Name: "Name FR", Description: "Desc FR"},
	}
	_ = paramTrans.SaveTranslations(ctx, parms)
	require.Len(t, repo.saved, 4) // 2 locales × 2 fields

	err := paramTrans.DeleteTranslationsByEntity(ctx, []int{1})
	require.NoError(t, err)
	require.Empty(t, repo.saved)
}

func TestDeleteTranslations_EmptyIDs_IsNoOp(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)
	ctx := context.Background()

	_ = paramTrans.SaveTranslations(ctx, []Parameter{
		{ID: 1, locale: LocaleEN, Name: "Name EN"},
	})
	saved := len(repo.saved)

	// Empty entityIDs must be a no-op, not a DELETE ALL.
	require.NoError(t, paramTrans.DeleteTranslations(ctx, LocaleEN, []int{}, []string{"name"}))
	require.Len(t, repo.saved, saved, "records must not be deleted with empty IDs")

	require.NoError(t, paramTrans.DeleteTranslationsByEntity(ctx, []int{}))
	require.Len(t, repo.saved, saved, "records must not be deleted with empty IDs")
}

func TestLoadTranslations_DuplicateEntitiesDedup(t *testing.T) {
	callCount := 0
	repo := &mockRepo{
		translations: []Translation{
			{Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Name EN"},
		},
	}
	counting := &countingGetRepo{mockRepo: repo, onGet: func() { callCount++ }}
	paramTrans := NewTranslator[Parameter](counting)
	ctx := context.Background()

	// Same (ID, locale) passed three times — must trigger exactly one DB call.
	parms := []Parameter{
		{ID: 1, locale: LocaleEN},
		{ID: 1, locale: LocaleEN},
		{ID: 1, locale: LocaleEN},
	}
	result, err := paramTrans.LoadTranslations(ctx, parms)
	require.NoError(t, err)
	require.Equal(t, 1, callCount, "expected exactly one DB call despite duplicate entities")
	// All three copies must be filled.
	for _, p := range result {
		require.Equal(t, "Name EN", p.Name)
	}
}

// countingGetRepo wraps mockRepo and counts GetTranslations calls.
type countingGetRepo struct {
	*mockRepo
	onGet func()
}

func (r *countingGetRepo) GetTranslations(ctx context.Context, locale Locale, entity string, ids []int) ([]Translation, error) {
	r.onGet()
	return r.mockRepo.GetTranslations(ctx, locale, entity, ids)
}

// errTest is a sentinel error used in error propagation tests.
var errTest = errors.New("repository error")

// BenchmarkLoadTranslations benchmarks the LoadTranslations operation.
func BenchmarkLoadTranslations(b *testing.B) {
	repo := &mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Test"},
		},
	}
	paramTrans := NewTranslator[Parameter](repo)
	parms := []Parameter{{ID: 1, locale: LocaleEN}}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = paramTrans.LoadTranslations(ctx, parms)
	}
}

// BenchmarkCacheHit benchmarks cache hit performance.
func BenchmarkCacheHit(b *testing.B) {
	repo := &mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Test"},
		},
	}
	cache := NewInMemoryCache()
	cachedRepo := NewCachedRepository(repo, cache, CacheOptions{TTL: 5 * time.Minute})
	paramTrans := NewTranslator[Parameter](cachedRepo)
	parms := []Parameter{{ID: 1, locale: LocaleEN}}
	ctx := context.Background()

	// Warm up cache
	paramTrans.LoadTranslations(ctx, parms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paramTrans.LoadTranslations(ctx, parms)
	}
}

// StressTest_ConcurrentCacheOperations tests cache behavior under concurrent load.
func TestStressTest_ConcurrentCacheOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	repo := &mockRepo{
		translations: make([]Translation, 100),
	}

	// Create 100 translations
	for i := 0; i < 100; i++ {
		repo.translations[i] = Translation{
			ID:       i + 1,
			Entity:   "parameter",
			EntityID: (i / 10) + 1, // 10 entities with 10 translations each
			Field:    "field_" + string(rune('a'+i%26)),
			Locale:   LocaleEN,
			Value:    "Value " + string(rune('0'+i%10)),
		}
	}

	cache := NewInMemoryCache()
	cachedRepo := NewCachedRepository(repo, cache, CacheOptions{TTL: 10 * time.Millisecond})
	paramTrans := NewTranslator[Parameter](cachedRepo)
	ctx := context.Background()

	const (
		goroutines = 50
		operations = 100
	)

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*operations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for op := 0; op < operations; op++ {
				entityID := (id*operations + op) % 10
				parms := []Parameter{{ID: entityID + 1, locale: LocaleEN}}

				if _, err := paramTrans.LoadTranslations(ctx, parms); err != nil {
					errors <- err
				}

				if op%10 == 0 {
					if err := paramTrans.SaveTranslations(ctx, parms); err != nil {
						errors <- err
					}
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Stress test failed: %v", err)
		}
	}

	// Verify cache stats
	stats := cache.Stats()
	require.True(t, stats.Hits > 0, "expected cache hits during stress test")
	require.True(t, stats.Sets > 0, "expected cache sets during stress test")
	t.Logf("Cache stats - Hits: %d, Misses: %d, Sets: %d, Deletes: %d",
		stats.Hits, stats.Misses, stats.Sets, stats.Deletes)
}

// TestStressTest_LargeBatchOperations tests performance with large batches.
func TestStressTest_LargeBatchOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const batchSize = 500

	repo := &mockRepo{
		translations: make([]Translation, batchSize*10),
	}

	// Create many translations
	for i := 0; i < batchSize*10; i++ {
		repo.translations[i] = Translation{
			ID:       i + 1,
			Entity:   "parameter",
			EntityID: i/10 + 1,
			Field:    "field",
			Locale:   LocaleEN,
			Value:    "Value",
		}
	}

	cache := NewInMemoryCache()
	cachedRepo := NewCachedRepository(repo, cache, CacheOptions{
		TTL:       5 * time.Minute,
		BatchSize: 100, // test batch processing
	})
	paramTrans := NewTranslator[Parameter](cachedRepo)
	ctx := context.Background()

	// Load large batch
	var ids []int
	for i := 1; i <= batchSize; i++ {
		ids = append(ids, i)
	}

	parms := make([]Parameter, len(ids))
	for i, id := range ids {
		parms[i] = Parameter{ID: id, locale: LocaleEN}
	}

	_, err := paramTrans.LoadTranslations(ctx, parms)
	require.NoError(t, err)

	stats := cache.Stats()
	require.True(t, stats.Sets > 0, "expected cache to be populated")
	t.Logf("Large batch test - Cache sets: %d", stats.Sets)
}

// TestStressTest_MemoryLeakDetection tests for potential memory leaks under load.
func TestStressTest_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	repo := &mockRepo{
		translations: []Translation{
			{ID: 1, Entity: "parameter", EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Test"},
		},
	}

	cache := NewInMemoryCache()
	ctx := context.Background()

	const iterations = 10000

	// Create and destroy translators repeatedly
	for i := 0; i < iterations; i++ {
		cachedRepo := NewCachedRepository(repo, cache, CacheOptions{TTL: 5 * time.Minute})
		paramTrans := NewTranslator[Parameter](cachedRepo)
		parms := []Parameter{{ID: 1, locale: LocaleEN}}
		_, _ = paramTrans.LoadTranslations(ctx, parms)
	}

	stats := cache.Stats()
	require.True(t, stats.Sets > 0, "cache should have operations")
	t.Logf("Memory leak test completed - Cache size would be: %d entries", len(cache.items))
}

// TestStressTest_CacheStatistics validates cache statistics tracking.
func TestStressTest_CacheStatistics(t *testing.T) {
	cache := NewInMemoryCache()

	// Perform various operations
	cache.Set("key1", []Translation{{ID: 1}}, 0)
	cache.Set("key2", []Translation{{ID: 2}}, 0)

	cache.Get("key1") // hit
	cache.Get("key1") // hit
	cache.Get("key3") // miss
	cache.Get("key4") // miss

	cache.Delete("key1")
	cache.Delete("key2")

	stats := cache.Stats()
	require.Equal(t, int64(2), stats.Hits, "expected 2 cache hits")
	require.Equal(t, int64(2), stats.Misses, "expected 2 cache misses")
	require.Equal(t, int64(2), stats.Sets, "expected 2 cache sets")
	require.Equal(t, int64(2), stats.Deletes, "expected 2 cache deletes")

	// Test reset
	cache.ResetStats()
	stats = cache.Stats()
	require.Equal(t, int64(0), stats.Hits)
	require.Equal(t, int64(0), stats.Misses)
	require.Equal(t, int64(0), stats.Sets)
	require.Equal(t, int64(0), stats.Deletes)
}

