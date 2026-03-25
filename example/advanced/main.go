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

// Article represents a complex entity with multiple translatable fields and associated metadata.
type Article struct {
	ID       int
	Title    string
	Content  string
	Summary  string
	locale   gotrans.Locale
	Author   string // not translatable
	CreateAt time.Time
	UpdateAt time.Time
}

// Implement Translatable interface
func (a Article) TranslationEntityLocale() gotrans.Locale { return a.locale }
func (a Article) TranslationEntityID() int                { return a.ID }
func (a Article) TranslatableFields() map[string]string {
	return map[string]string{
		"Title":   "title",
		"Content": "content",
		"Summary": "summary",
	}
}
func (a Article) TranslationEntityName() string { return "article" }

// PageManager handles article translations across multiple pages/locales.
type PageManager struct {
	translator gotrans.Translator[Article]
	db         *sqlx.DB
	cache      gotrans.TranslationCache
}

// NewPageManager creates a new page manager with advanced configuration.
func NewPageManager(db *sqlx.DB) *PageManager {
	repo := mysql.NewTranslationRepository(db)
	cache := gotrans.NewInMemoryCache()

	cachedRepo := gotrans.NewCachedRepository(repo, cache, gotrans.CacheOptions{
		TTL:                   10 * time.Minute,
		BatchSize:             200,
		DefaultContextTimeout: 10 * time.Second,
	})

	translator := gotrans.NewTranslator[Article](cachedRepo)

	return &PageManager{
		translator: translator,
		db:         db,
		cache:      cache,
	}
}

// PublishArticles publishes multiple articles with translations for different locales.
func (pm *PageManager) PublishArticles(ctx context.Context, articleData map[string]map[gotrans.Locale]Article) error {
	for articleName, locales := range articleData {
		var articles []Article
		for _, article := range locales {
			articles = append(articles, article)
		}

		if err := pm.translator.SaveTranslations(ctx, articles); err != nil {
			return fmt.Errorf("failed to publish %s: %w", articleName, err)
		}
		fmt.Printf("✓ Published '%s' in %d languages\n", articleName, len(locales))
	}
	return nil
}

// GetArticlesByLocale retrieves articles for a specific locale.
func (pm *PageManager) GetArticlesByLocale(ctx context.Context, locale gotrans.Locale, articleIDs []int) ([]Article, error) {
	articles := make([]Article, len(articleIDs))
	for i, id := range articleIDs {
		articles[i] = Article{ID: id, locale: locale}
	}

	loaded, err := pm.translator.LoadTranslations(ctx, articles)
	if err != nil {
		return nil, err
	}
	return loaded, nil
}

// GetCacheStats returns current cache statistics.
func (pm *PageManager) GetCacheStats() gotrans.CacheStats {
	return pm.cache.Stats()
}

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

	// Create page manager
	pm := NewPageManager(db)

	fmt.Println("=== Advanced Multi-Locale Example ===")

	// Create articles with multiple locale translations
	articlesData := map[string]map[gotrans.Locale]Article{
		"Getting Started": {
			gotrans.LocaleEN: {
				ID:      1,
				Title:   "Getting Started with gotrans",
				Content: "Learn the basics of gotrans...",
				Summary: "An introduction to gotrans",
				locale:  gotrans.LocaleEN,
				Author:  "John Doe",
			},
			gotrans.LocaleFR: {
				ID:      1,
				Title:   "Commencer avec gotrans",
				Content: "Apprenez les bases de gotrans...",
				Summary: "Une introduction à gotrans",
				locale:  gotrans.LocaleFR,
				Author:  "John Doe",
			},
			gotrans.LocaleDE: {
				ID:      1,
				Title:   "Erste Schritte mit gotrans",
				Content: "Lernen Sie die Grundlagen von gotrans...",
				Summary: "Eine Einführung in gotrans",
				locale:  gotrans.LocaleDE,
				Author:  "John Doe",
			},
		},
		"Advanced Features": {
			gotrans.LocaleEN: {
				ID:      2,
				Title:   "Advanced Features of gotrans",
				Content: "Explore advanced caching and performance...",
				Summary: "Advanced usage patterns",
				locale:  gotrans.LocaleEN,
				Author:  "Jane Smith",
			},
			gotrans.LocaleFR: {
				ID:      2,
				Title:   "Fonctionnalités avancées de gotrans",
				Content: "Explorez la mise en cache avancée et les performances...",
				Summary: "Modèles d'utilisation avancés",
				locale:  gotrans.LocaleFR,
				Author:  "Jane Smith",
			},
		},
	}

	// Publish all articles
	fmt.Println("\n--- Publishing Articles ---")
	if err := pm.PublishArticles(ctx, articlesData); err != nil {
		log.Fatalf("failed to publish articles: %v", err)
	}

	// Retrieve articles by locale
	fmt.Println("\n--- Retrieving Articles ---")

	// Get English articles
	fmt.Println("\nEnglish Articles:")
	articles, err := pm.GetArticlesByLocale(ctx, gotrans.LocaleEN, []int{1, 2})
	if err != nil {
		log.Fatalf("failed to get articles: %v", err)
	}

	for _, a := range articles {
		fmt.Printf("  • %s (%s)\n", a.Title, a.locale.Name())
	}

	// Get French articles
	fmt.Println("\nFrench Articles:")
	articles, err = pm.GetArticlesByLocale(ctx, gotrans.LocaleFR, []int{1, 2})
	if err != nil {
		log.Fatalf("failed to get articles: %v", err)
	}

	for _, a := range articles {
		fmt.Printf("  • %s (%s)\n", a.Title, a.locale.Name())
	}

	// Get German articles
	fmt.Println("\nGerman Articles:")
	articles, err = pm.GetArticlesByLocale(ctx, gotrans.LocaleDE, []int{1, 2})
	if err != nil {
		log.Fatalf("failed to get articles: %v", err)
	}

	for _, a := range articles {
		fmt.Printf("  • %s (%s)\n", a.Title, a.locale.Name())
	}

	// Show cache statistics
	fmt.Println("\n--- Cache Statistics ---")
	stats := pm.GetCacheStats()
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		hitRate := float64(stats.Hits) / float64(totalRequests) * 100
		fmt.Printf("Total Requests: %d\n", totalRequests)
		fmt.Printf("Hits: %d (%.1f%%)\n", stats.Hits, hitRate)
		fmt.Printf("Misses: %d\n", stats.Misses)
		fmt.Printf("Sets: %d\n", stats.Sets)
	}

	// Demonstrate context timeout
	fmt.Println("\n--- Context Timeout Example ---")
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	articles, err = pm.GetArticlesByLocale(timeoutCtx, gotrans.LocaleEN, []int{1})
	if err == nil {
		fmt.Println("✓ Request completed within timeout")
	} else {
		fmt.Printf("✗ Request failed: %v\n", err)
	}

	fmt.Println("\n✓ Advanced example completed successfully")
}

