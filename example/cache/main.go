package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ivan-gorbushko/gotrans"
	"github.com/ivan-gorbushko/gotrans/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Product struct {
	ID          int
	Title       string
	Description string
	locale      gotrans.Locale
}

func (p Product) TranslationEntityLocale() gotrans.Locale { return p.locale }
func (p Product) TranslationEntityID() int                { return p.ID }
func (p Product) TranslationEntityName() string           { return "product" }
func (p Product) TranslatableFields() map[string]string {
	return map[string]string{
		"Title":       "title",
		"Description": "description",
	}
}

func main() {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE translations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity TEXT,
			entity_id INTEGER,
			field TEXT,
			locale TEXT,
			value TEXT,
			UNIQUE(entity, entity_id, field, locale)
		)
	`)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	// --- Without cache ---
	repo := mysql.NewTranslationRepository(db)

	// --- With in-memory cache (TTL = 30 seconds) ---
	cachedRepo := gotrans.NewCachedRepositoryInMemory(repo, gotrans.CacheOptions{
		TTL: 30 * time.Second,
	})

	translator := gotrans.NewTranslator[Product](cachedRepo)

	// Seed some data
	products := []Product{
		{ID: 1, locale: gotrans.LocaleEN, Title: "Apple", Description: "Fresh fruit"},
		{ID: 2, locale: gotrans.LocaleEN, Title: "Banana", Description: "Yellow fruit"},
		{ID: 1, locale: gotrans.LocaleFR, Title: "Pomme", Description: "Fruit frais"},
		{ID: 2, locale: gotrans.LocaleFR, Title: "Banane", Description: "Fruit jaune"},
	}
	if err = translator.SaveTranslations(ctx, products); err != nil {
		log.Fatalf("failed to seed: %v", err)
	}
	fmt.Println("Seeded translations for EN and FR.")
	fmt.Println()

	// --- Example 1: First load — cache miss, goes to DB ---
	fmt.Println("=== Example 1: First Load (cache miss → DB) ===")
	toLoad := []Product{
		{ID: 1, locale: gotrans.LocaleEN},
		{ID: 2, locale: gotrans.LocaleEN},
	}
	loaded, _ := translator.LoadTranslations(ctx, toLoad)
	for _, p := range loaded {
		fmt.Printf("  Product %d (EN): %s — %s\n", p.ID, p.Title, p.Description)
	}

	// --- Example 2: Second load — cache hit, no DB call ---
	fmt.Println("\n=== Example 2: Second Load (cache hit → no DB) ===")
	loaded, _ = translator.LoadTranslations(ctx, toLoad)
	for _, p := range loaded {
		fmt.Printf("  Product %d (EN): %s — %s\n", p.ID, p.Title, p.Description)
	}
	fmt.Println("  (served from cache)")

	// --- Example 3: Partial cache hit ---
	// ID 1 is cached, ID 3 is not → only ID 3 goes to DB
	fmt.Println("\n=== Example 3: Partial Cache Hit ===")
	partial := []Product{
		{ID: 1, locale: gotrans.LocaleEN}, // cached
		{ID: 3, locale: gotrans.LocaleEN}, // not in cache, not in DB either
	}
	loaded, _ = translator.LoadTranslations(ctx, partial)
	fmt.Printf("  Loaded %d product(s) — ID 1 from cache, ID 3 from DB\n", len(loaded))
	for _, p := range loaded {
		fmt.Printf("  Product %d (EN): %q %q\n", p.ID, p.Title, p.Description)
	}

	// --- Example 4: Cache invalidation on update ---
	fmt.Println("\n=== Example 4: Cache Invalidation on Update ===")
	updated := []Product{
		{ID: 1, locale: gotrans.LocaleEN, Title: "Green Apple", Description: "Freshly picked"},
	}
	if err = translator.SaveTranslations(ctx, updated); err != nil {
		log.Fatalf("failed to update: %v", err)
	}
	fmt.Println("  Updated product 1 — cache entry invalidated automatically.")

	reloaded, _ := translator.LoadTranslations(ctx, []Product{{ID: 1, locale: gotrans.LocaleEN}})
	for _, p := range reloaded {
		fmt.Printf("  Product %d (EN): %s — %s\n", p.ID, p.Title, p.Description)
	}

	// --- Example 5: Cache invalidation on delete (specific locale) ---
	fmt.Println("\n=== Example 5: Cache Invalidation on Delete (EN only) ===")
	_ = translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", []int{2}, []string{"title", "description"})
	fmt.Println("  Deleted EN translations for product 2 — cache invalidated.")

	// FR cache for product 2 is still valid
	frLoaded, _ := translator.LoadTranslations(ctx, []Product{{ID: 2, locale: gotrans.LocaleFR}})
	for _, p := range frLoaded {
		fmt.Printf("  Product %d (FR) still cached: %s — %s\n", p.ID, p.Title, p.Description)
	}

	// --- Example 6: Delete all locales — cross-locale cache eviction ---
	fmt.Println("\n=== Example 6: Delete All Locales (cross-locale eviction) ===")
	_ = translator.DeleteTranslationsByEntity(ctx, "product", []int{1})
	fmt.Println("  Deleted all translations for product 1 — all locale cache entries evicted.")

	// Both EN and FR must reload from DB (empty now)
	enLoaded, _ := translator.LoadTranslations(ctx, []Product{{ID: 1, locale: gotrans.LocaleEN}})
	frLoaded2, _ := translator.LoadTranslations(ctx, []Product{{ID: 1, locale: gotrans.LocaleFR}})
	fmt.Printf("  Product 1 EN title after delete: %q\n", enLoaded[0].Title)
	fmt.Printf("  Product 1 FR title after delete: %q\n", frLoaded2[0].Title)

	// --- Example 7: TTL expiry ---
	fmt.Println("\n=== Example 7: Custom Cache with Short TTL ===")
	shortTTLRepo := gotrans.NewCachedRepositoryInMemory(repo, gotrans.CacheOptions{
		TTL: 50 * time.Millisecond,
	})
	shortTranslator := gotrans.NewTranslator[Product](shortTTLRepo)

	// Re-seed product 2 EN
	_ = shortTranslator.SaveTranslations(ctx, []Product{
		{ID: 2, locale: gotrans.LocaleEN, Title: "Banana", Description: "Yellow fruit"},
	})

	first, _ := shortTranslator.LoadTranslations(ctx, []Product{{ID: 2, locale: gotrans.LocaleEN}})
	fmt.Printf("  Before TTL expires: %q\n", first[0].Title)

	time.Sleep(80 * time.Millisecond)

	second, _ := shortTranslator.LoadTranslations(ctx, []Product{{ID: 2, locale: gotrans.LocaleEN}})
	fmt.Printf("  After TTL expires (reloaded from DB): %q\n", second[0].Title)

	// --- Example 8: Custom cache backend ---
	fmt.Println("\n=== Example 8: Custom Cache Backend ===")
	customCache := gotrans.NewInMemoryCache() // swap this for Redis, Memcached, etc.
	customCachedRepo := gotrans.NewCachedRepository(repo, customCache, gotrans.CacheOptions{
		TTL: 10 * time.Minute,
	})
	customTranslator := gotrans.NewTranslator[Product](customCachedRepo)

	res, _ := customTranslator.LoadTranslations(ctx, []Product{{ID: 2, locale: gotrans.LocaleEN}})
	fmt.Printf("  Loaded via custom cache: %q\n", res[0].Title)
	customCache.Clear()
	fmt.Println("  Cache cleared manually via custom cache handle.")
}

