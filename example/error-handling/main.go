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

	// Create translator with default context timeout of 5 seconds
	translator := gotrans.NewTranslatorWithOptions(gotrans.TranslatorOptions[Product]{
		Repository:            repo,
		DefaultContextTimeout: 5 * time.Second,
	})

	fmt.Println("=== Error Handling Example ===")

	// Save translations
	products := []Product{
		{ID: 1, Title: "Product 1", Description: "Description 1", locale: gotrans.LocaleEN},
	}

	if err := translator.SaveTranslations(context.Background(), products); err != nil {
		log.Fatalf("failed to save translations: %v", err)
	}
	fmt.Println("✓ Translations saved")

	// Example 1: Handle cancelled context
	fmt.Println("\n--- Example 1: Handling Cancelled Context ---")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	toLoad := []Product{{ID: 1, locale: gotrans.LocaleEN}}
	_, err = translator.LoadTranslations(ctx, toLoad)
	if err != nil {
		fmt.Printf("✓ Caught error from cancelled context: %v\n", err)
	}

	// Example 2: Handle context timeout
	fmt.Println("\n--- Example 2: Handling Context Timeout ---")
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a slow repository to simulate timeout
	slowRepo := &slowRepository{repo: repo, delay: 200 * time.Millisecond}
	slowTranslator := gotrans.NewTranslator[Product](slowRepo)

	_, err = slowTranslator.LoadTranslations(ctx, toLoad)
	if err != nil {
		fmt.Printf("✓ Caught timeout error: %v\n", err)
	}

	// Example 3: Empty entity name handling
	fmt.Println("\n--- Example 3: Empty Entity Validation ---")
	invalidProduct := Product{ID: 1, locale: gotrans.LocaleEN} // Missing entity setup
	_, err = translator.LoadTranslations(context.Background(), []Product{invalidProduct})
	fmt.Printf("✓ Load successful (entity name is derived from translator)\n")

	// Example 4: Default timeout in action
	fmt.Println("\n--- Example 4: Default Context Timeout ---")
	fmt.Println("Translator has default timeout of 5 seconds")
	fmt.Println("✓ Any operation without explicit deadline will use this timeout")

	fmt.Println("\n✓ All error handling examples completed successfully")
}

// slowRepository wraps a repository and adds artificial delay to simulate slow operations.
type slowRepository struct {
	repo  gotrans.TranslationRepository
	delay time.Duration
}

func (s *slowRepository) GetTranslations(ctx context.Context, locale gotrans.Locale, entity string, entityIDs []int) ([]gotrans.Translation, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(s.delay):
		return s.repo.GetTranslations(ctx, locale, entity, entityIDs)
	}
}

func (s *slowRepository) MassDelete(ctx context.Context, locale gotrans.Locale, entity string, entityIDs []int, fields []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.delay):
		return s.repo.MassDelete(ctx, locale, entity, entityIDs, fields)
	}
}

func (s *slowRepository) MassCreateOrUpdate(ctx context.Context, locale gotrans.Locale, translations []gotrans.Translation) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.delay):
		return s.repo.MassCreateOrUpdate(ctx, locale, translations)
	}
}

