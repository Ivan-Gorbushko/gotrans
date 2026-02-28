# gotrans
This repository provides a lightweight, framework-agnostic translation module for Golang applications. It is designed to manage multi-language content directly within backend business logic, without relying on heavy external localization frameworks.

## Concept
- Translatable fields in your entities are plain string fields (not map or struct).
- Each translation operation (load/save/delete) works with a single language (locale) per call.
- You explicitly associate struct fields with translation field IDs via the Translatable interface.
- The repository stores translations in a normalized table (see below).

## Entity example
```go
// Example entity
 type Product struct {
     ID          int
     Title       string
     Description string
 }

 func (p Product) TranslationEntityID() int { return p.ID }
 func (p Product) TranslatableFieldMap() map[string]string {
     return map[string]string{
         "Title":       "title",
         "Description": "description",
     }
 }
```

## Usage example
```go
repo := mysql.NewTranslationRepository(db) // db: *sqlx.DB (MySQL or SQLite)
translator := gotrans.NewTranslator[Product](repo)
ctx := context.Background()

// Save translation
product := Product{ID: 1, Title: "Apple", Description: "Fresh fruit"}
_ = translator.SaveTranslations(ctx, gotrans.LocaleEN, []Product{product})

// Load translation
product.Title = ""
product.Description = ""
products, _ := translator.LoadTranslations(ctx, gotrans.LocaleEN, []Product{product})
fmt.Printf("Loaded: Title=%s, Description=%s\n", products[0].Title, products[0].Description)

// Delete translation
_ = translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", []int{1}, []string{"title", "description"})
```

## Translatable interface
```go
type Translatable interface {
    TranslationEntityID() int
    TranslatableFieldMap() map[string]string // key: struct field name, value: translation field id
}
```

## Mysql table structure
```sql
CREATE TABLE IF NOT EXISTS translations (
    id BIGINT AUTO_INCREMENT,
    entity VARCHAR(100) NOT NULL,
    entity_id BIGINT NOT NULL,
    field VARCHAR(100) NOT NULL,
    locale VARCHAR(10) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uniq_translation (entity, entity_id, field, locale)
)
COLLATE = utf8mb4_unicode_ci;
```

## Features
- Works with any SQL database (MySQL, SQLite, etc.) via sqlx.
- No reflection for translation logic at runtime.
- Explicit field mapping for maximum safety and flexibility.
- Easy integration with any Go project.

## Example with SQLite
See `go run example/main.go` for a full working demo with SQLite.
