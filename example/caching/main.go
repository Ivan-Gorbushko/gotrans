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

// Product represents an entity with translatable fields.
type Product struct {
	ID          int
	Title       string
	Description string
	locale      gotrans.Locale
}

// Implement Translatable interface
func (p Product) TranslationEntityLocale() gotrans.Locale { return p.locale }
func (p Product) TranslationEntityID() int                { return p.ID }
func (p Product) TranslatableFields() map[string]string {
	return map[string]string{
		"Title":       "title",
		"Description": "description",
	}
}
func (p Product) TranslationEntityName() string { return "product" }

func main() {
	ctx := context.Background()

	// Open in-memory SQLite DB
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create translations table
	_, err = db.Exec(`
		CREATE TABLE translations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity TEXT,
			entity_id INTEGER,
			field TEXT,
			locale TEXT,
			value TEXT
		)
	`)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	// Create repository
	repo := mysql.NewTranslationRepository(db)

	// Create in-memory cache with 5 minute TTL
	cache := gotrans.NewInMemoryCache()

	// Create cached repository
	cachedRepo := gotrans.NewCachedRepository(repo, cache, gotrans.CacheOptions{
		TTL:       5 * time.Minute,
		BatchSize: 100,
	})

	// Create translator with cached repository
	translator := gotrans.NewTranslator[Product](cachedRepo)

	fmt.Println("=== Caching Example ===")

	// Save some translations
	products := []Product{
		{ID: 1, Title: "Product 1 EN", Description: "Description 1", locale: gotrans.LocaleEN},
		{ID: 2, Title: "Product 2 EN", Description: "Description 2", locale: gotrans.LocaleEN},
		{ID: 3, Title: "Product 3 EN", Description: "Description 3", locale: gotrans.LocaleEN},
	}

	if err := translator.SaveTranslations(ctx, products); err != nil {
		log.Fatalf("failed to save translations: %v", err)
	}
	fmt.Println("Saved 3 products")

	// First load - will hit the database
	fmt.Println("\n--- First Load (Cache Miss) ---")
	toLoad := []Product{
		{ID: 1, locale: gotrans.LocaleEN},
		{ID: 2, locale: gotrans.LocaleEN},
		{ID: 3, locale: gotrans.LocaleEN},
	}

	loaded, err := translator.LoadTranslations(ctx, toLoad)
	if err != nil {
		log.Fatalf("failed to load translations: %v", err)
	}

	stats := cache.Stats()
	fmt.Printf("Loaded %d products\n", len(loaded))
	fmt.Printf("Cache Stats - Hits: %d, Misses: %d, Sets: %d\n", stats.Hits, stats.Misses, stats.Sets)

	// Second load - will use cache
	fmt.Println("\n--- Second Load (Cache Hit) ---")
	toLoad = []Product{
		{ID: 1, locale: gotrans.LocaleEN},
		{ID: 2, locale: gotrans.LocaleEN},
	}

	loaded, err = translator.LoadTranslations(ctx, toLoad)
	if err != nil {
		log.Fatalf("failed to load translations: %v", err)
	}

	stats = cache.Stats()
	fmt.Printf("Loaded %d products from cache\n", len(loaded))
	fmt.Printf("Cache Stats - Hits: %d, Misses: %d, Sets: %d\n", stats.Hits, stats.Misses, stats.Sets)

	// Mixed load - some from cache, some from database
	fmt.Println("\n--- Mixed Load (Partial Cache Hit) ---")
	toLoad = []Product{
		{ID: 1, locale: gotrans.LocaleEN},
		{ID: 4, locale: gotrans.LocaleEN},
	}

	loaded, err = translator.LoadTranslations(ctx, toLoad)
	if err != nil {
		log.Fatalf("failed to load translations: %v", err)
	}

	stats = cache.Stats()
	fmt.Printf("Loaded %d products (1 from cache, 1 from DB)\n", len(loaded))
	fmt.Printf("Cache Stats - Hits: %d, Misses: %d, Sets: %d\n", stats.Hits, stats.Misses, stats.Sets)

	// Print cache hit rate
	fmt.Println("\n--- Cache Performance ---")
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		hitRate := float64(stats.Hits) / float64(totalRequests) * 100
		fmt.Printf("Hit Rate: %.1f%% (Hits: %d, Misses: %d)\n", hitRate, stats.Hits, stats.Misses)
	}
}

