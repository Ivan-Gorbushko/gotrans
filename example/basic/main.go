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

	repo := mysql.NewTranslationRepository(db)
	translator := gotrans.NewTranslator[Product](repo)

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

	// Load translations for English
	fmt.Println("=== English Products ===")
	enProducts := []Product{
		{ID: 1, locale: gotrans.LocaleEN},
		{ID: 2, locale: gotrans.LocaleEN},
	}
	enProducts, err = translator.LoadTranslations(ctx, enProducts)
	if err != nil {
		log.Fatalf("failed to load EN: %v", err)
	}
	for _, p := range enProducts {
		fmt.Printf("  %d. %s - %s\n", p.ID, p.Title, p.Description)
	}
	fmt.Println()

	// Load translations for French
	fmt.Println("=== French Products ===")
	frProducts := []Product{
		{ID: 1, locale: gotrans.LocaleFR},
		{ID: 2, locale: gotrans.LocaleFR},
	}
	frProducts, err = translator.LoadTranslations(ctx, frProducts)
	if err != nil {
		log.Fatalf("failed to load FR: %v", err)
	}
	for _, p := range frProducts {
		fmt.Printf("  %d. %s - %s\n", p.ID, p.Title, p.Description)
	}
	fmt.Println()

	// Update translations
	fmt.Println("=== Update Translation ===")
	updateProducts := []Product{
		{ID: 1, locale: gotrans.LocaleEN, Title: "Apple (Updated)", Description: "Fresh fruit (Updated)"},
	}
	if err = translator.SaveTranslations(ctx, updateProducts); err != nil {
		log.Fatalf("failed to update: %v", err)
	}

	// Reload to verify update
	reloadProducts := []Product{
		{ID: 1, locale: gotrans.LocaleEN},
	}
	reloadProducts, err = translator.LoadTranslations(ctx, reloadProducts)
	if err != nil {
		log.Fatalf("failed to reload: %v", err)
	}
	fmt.Printf("Updated: %s - %s\n", reloadProducts[0].Title, reloadProducts[0].Description)
	fmt.Println()

	// Delete translation
	fmt.Println("=== Delete Translation ===")
	if err = translator.DeleteTranslations(ctx, gotrans.LocaleEN, []int{1}, []string{"title"}); err != nil {
		log.Fatalf("failed to delete: %v", err)
	}
	fmt.Println("Deleted title translation for product 1 (EN)")
}

