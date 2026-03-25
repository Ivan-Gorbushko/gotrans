package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ivan-gorbushko/gotrans"
	"github.com/ivan-gorbushko/gotrans/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Example entity with built-in locale
type Product struct {
	ID          int
	Locale      gotrans.Locale
	Title       string
	Description string
}

// Implement Translatable interface
func (p Product) TranslationLocale() gotrans.Locale { return p.Locale }
func (p Product) TranslationEntityID() int          { return p.ID }
func (p Product) TranslatableFields() map[string]string {
	return map[string]string{
		"Title":       "title",
		"Description": "description",
	}
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
			value TEXT,
			UNIQUE(entity, entity_id, field, locale)
		)
	`)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	repo := mysql.NewTranslationRepository(db)
	translator := gotrans.NewTranslator[Product](repo)

	// Example 1: Save translations for English
	fmt.Println("=== Example 1: Save English Translations ===")
	products := []Product{
		{ID: 1, Locale: gotrans.LocaleEN, Title: "Apple", Description: "Fresh fruit"},
		{ID: 2, Locale: gotrans.LocaleEN, Title: "Banana", Description: "Yellow fruit"},
	}
	err = translator.SaveTranslations(ctx, products)
	if err != nil {
		log.Fatalf("failed to save: %v", err)
	}

	// Example 2: Save translations for French (demonstrates grouping by locale)
	fmt.Println("\n=== Example 2: Save French Translations ===")
	productsFR := []Product{
		{ID: 1, Locale: gotrans.LocaleFR, Title: "Pomme", Description: "Fruit frais"},
		{ID: 2, Locale: gotrans.LocaleFR, Title: "Banane", Description: "Fruit jaune"},
	}
	err = translator.SaveTranslations(ctx, productsFR)
	if err != nil {
		log.Fatalf("failed to save: %v", err)
	}

	fmt.Println("\nSaved translations:")
	rows, _ := db.Queryx("SELECT entity, entity_id, field, locale, value FROM translations ORDER BY locale, entity_id, field")
	for rows.Next() {
		var entity string
		var entityID int
		var field, locale, value string
		_ = rows.Scan(&entity, &entityID, &field, &locale, &value)
		fmt.Printf("  %s[%d].%s[%s] = %s\n", entity, entityID, field, locale, value)
	}

	// Example 3: Load translations for English
	fmt.Println("\n=== Example 3: Load English Translations ===")
	productsToLoad := []Product{
		{ID: 1, Locale: gotrans.LocaleEN},
		{ID: 2, Locale: gotrans.LocaleEN},
	}
	productsLoaded, _ := translator.LoadTranslations(ctx, productsToLoad)
	for _, p := range productsLoaded {
		fmt.Printf("Product %d (EN): %s - %s\n", p.ID, p.Title, p.Description)
	}

	// Example 4: Load translations for French
	fmt.Println("\n=== Example 4: Load French Translations ===")
	productsToLoadFR := []Product{
		{ID: 1, Locale: gotrans.LocaleFR},
		{ID: 2, Locale: gotrans.LocaleFR},
	}
	productsLoadedFR, _ := translator.LoadTranslations(ctx, productsToLoadFR)
	for _, p := range productsLoadedFR {
		fmt.Printf("Product %d (FR): %s - %s\n", p.ID, p.Title, p.Description)
	}

	// Example 5: Delete translations for specific locale
	fmt.Println("\n=== Example 5: Delete English Translations ===")
	_ = translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", []int{1}, []string{"title", "description"})
	
	fmt.Println("Remaining translations:")
	rows, _ = db.Queryx("SELECT entity, entity_id, field, locale, value FROM translations ORDER BY locale, entity_id")
	count := 0
	for rows.Next() {
		var entity string
		var entityID int
		var field, locale, value string
		_ = rows.Scan(&entity, &entityID, &field, &locale, &value)
		fmt.Printf("  %s[%d].%s[%s] = %s\n", entity, entityID, field, locale, value)
		count++
	}
	if count == 0 {
		fmt.Println("  (no translations)")
	}

	// Example 6: Delete all translations for entity
	fmt.Println("\n=== Example 6: Delete All Translations for Entities ===")
	_ = translator.DeleteTranslationsByEntity(ctx, "product", []int{1, 2})
	
	fmt.Println("After delete all:")
	rows, _ = db.Queryx("SELECT COUNT(*) FROM translations")
	var count64 int64
	for rows.Next() {
		_ = rows.Scan(&count64)
		fmt.Printf("  Total translations: %d\n", count64)
	}
}

