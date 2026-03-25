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

	// Create in-memory cache with batch size control
	cache := gotrans.NewInMemoryCache()

	// Create cached repository with custom batch size for large datasets
	cachedRepo := gotrans.NewCachedRepository(repo, cache, gotrans.CacheOptions{
		TTL:       5 * time.Minute,
		BatchSize: 500, // Process 500 IDs per batch
	})

	translator := gotrans.NewTranslator[Product](cachedRepo)

	fmt.Println("=== Performance Example ===")

	// Create a large dataset
	const productCount = 1000
	fmt.Printf("Creating %d products...\n", productCount)

	var products []Product
	for i := 1; i <= productCount; i++ {
		products = append(products, Product{
			ID:          i,
			Title:       fmt.Sprintf("Product %d EN", i),
			Description: fmt.Sprintf("Description for product %d", i),
			locale:      gotrans.LocaleEN,
		})
	}

	// Save all products
	start := time.Now()
	if err := translator.SaveTranslations(ctx, products); err != nil {
		log.Fatalf("failed to save translations: %v", err)
	}
	fmt.Printf("✓ Saved %d products in %v\n", productCount, time.Since(start))

	// Load all products - first time (database hit)
	fmt.Println("\n--- Load All Products (First Time - Database) ---")
	toLoad := make([]Product, productCount)
	for i := 0; i < productCount; i++ {
		toLoad[i] = Product{ID: i + 1, locale: gotrans.LocaleEN}
	}

	start = time.Now()
	loaded, err := translator.LoadTranslations(ctx, toLoad)
	if err != nil {
		log.Fatalf("failed to load translations: %v", err)
	}
	duration := time.Since(start)

	stats := cache.Stats()
	fmt.Printf("✓ Loaded %d products in %v\n", len(loaded), duration)
	fmt.Printf("  Cache: Hits=%d, Misses=%d, Sets=%d, Batches=%d\n",
		stats.Hits, stats.Misses, stats.Sets, (productCount+499)/500)

	// Load all products - second time (cache hit)
	fmt.Println("\n--- Load All Products (Second Time - Cache) ---")
	cache.ResetStats()

	start = time.Now()
	loaded, err = translator.LoadTranslations(ctx, toLoad)
	if err != nil {
		log.Fatalf("failed to load translations: %v", err)
	}
	duration = time.Since(start)

	stats = cache.Stats()
	fmt.Printf("✓ Loaded %d products in %v (from cache)\n", len(loaded), duration)
	fmt.Printf("  Cache: Hits=%d, Misses=%d\n", stats.Hits, stats.Misses)

	// Load partial data
	fmt.Println("\n--- Load Partial Data (250 products) ---")
	cache.ResetStats()

	start = time.Now()
	partialLoad := make([]Product, 250)
	for i := 0; i < 250; i++ {
		partialLoad[i] = Product{ID: i + 1, locale: gotrans.LocaleEN}
	}
	loaded, err = translator.LoadTranslations(ctx, partialLoad)
	if err != nil {
		log.Fatalf("failed to load translations: %v", err)
	}
	duration = time.Since(start)

	stats = cache.Stats()
	fmt.Printf("✓ Loaded %d products in %v (from cache)\n", len(loaded), duration)
	fmt.Printf("  Cache: Hits=%d, Misses=%d\n", stats.Hits, stats.Misses)

	// Benchmark batch processing
	fmt.Println("\n--- Batch Processing Performance ---")
	fmt.Println("BatchSize: 500 IDs per database query")
	fmt.Printf("Total Products: %d\n", productCount)
	fmt.Printf("Number of Batches: %d\n", (productCount+499)/500)

	// Cache statistics summary
	fmt.Println("\n--- Final Cache Statistics ---")
	finalStats := cache.Stats()
	totalRequests := finalStats.Hits + finalStats.Misses
	if totalRequests > 0 {
		hitRate := float64(finalStats.Hits) / float64(totalRequests) * 100
		fmt.Printf("Total Requests: %d\n", totalRequests)
		fmt.Printf("Total Hits: %d\n", finalStats.Hits)
		fmt.Printf("Total Misses: %d\n", finalStats.Misses)
		fmt.Printf("Hit Rate: %.1f%%\n", hitRate)
		fmt.Printf("Total Sets: %d\n", finalStats.Sets)
	}

	fmt.Println("\n✓ Performance example completed")
}

