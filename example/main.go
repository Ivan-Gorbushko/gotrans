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

// Example entity
type Product struct {
	ID          int
	Title       string
	Description string

	metaTransFields map[string]string
}

func (p Product) TranslationEntityID() int { return p.ID }
func (p Product) TranslatableFieldMap() map[string]string {
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
			value TEXT
		)
	`)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	repo := mysql.NewTranslationRepository(db)
	translator := gotrans.NewTranslator[Product](repo)

	// Save translation
	product := Product{ID: 1, Title: "Apple", Description: "Fresh fruit"}
	_ = translator.SaveTranslations(ctx, gotrans.LocaleEN, []Product{product})

	fmt.Println("Saved translations:")
	rows, _ := db.Queryx("SELECT entity, entity_id, field, locale, value FROM translations")
	for rows.Next() {
		var entity string
		var entityID int
		var field, locale, value string
		_ = rows.Scan(&entity, &entityID, &field, &locale, &value)
		fmt.Printf("%s %d %s %s %s\n", entity, entityID, field, locale, value)
	}

	// Load translation
	product.Title = ""
	product.Description = ""
	products, _ := translator.LoadTranslations(ctx, gotrans.LocaleEN, []Product{product})
	fmt.Printf("Loaded: Title=%s, Description=%s\n", products[0].Title, products[0].Description)

	// Delete translation
	_ = translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", []int{1}, []string{"title", "description"})
	fmt.Println("After delete:")
	rows, _ = db.Queryx("SELECT entity, entity_id, field, locale, value FROM translations")
	for rows.Next() {
		var entity string
		var entityID int
		var field, locale, value string
		_ = rows.Scan(&entity, &entityID, &field, &locale, &value)
		fmt.Printf("%s %d %s %s %s\n", entity, entityID, field, locale, value)
	}
}
