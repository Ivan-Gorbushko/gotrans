# gotrans

Lightweight, framework-agnostic translation module for Go applications. Manage multi-language content directly within your backend business logic.

## Key Features

- **Embedded Locale**: Each entity carries its own locale information
- **Automatic Optimization**: Translations grouped by locale for efficient batch operations
- **Type Safe**: Uses Go generics for compile-time type checking
- **Explicit Field Mapping**: Clear association between struct fields and translation field IDs
- **Framework Agnostic**: Works with MySQL, SQLite, PostgreSQL, and any database supported by sqlx
- **41 Supported Languages**: Complete ISO-639-1 locale support
- **Zero Dependencies**: Only requires sqlx for database operations

## Installation

```bash
go get github.com/ivan-gorbushko/gotrans
```

## Quick Start

### 1. Define Your Entity

```go
type Product struct {
    ID          int
    Locale      gotrans.Locale  // Required: entity's language
    Title       string          // Translatable field
    Description string          // Translatable field
}

// Implement Translatable interface
func (p Product) TranslationLocale() gotrans.Locale {
    return p.Locale
}

func (p Product) TranslationEntityID() int {
    return p.ID
}

func (p Product) TranslatableFields() map[string]string {
    return map[string]string{
        "Title":       "title",
        "Description": "description",
    }
}
```

### 2. Setup Repository and Translator

```go
import (
    "github.com/ivan-gorbushko/gotrans"
    "github.com/ivan-gorbushko/gotrans/mysql"
    "github.com/jmoiron/sqlx"
)

db := sqlx.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
repo := mysql.NewTranslationRepository(db)
translator := gotrans.NewTranslator[Product](repo)
```

### 3. Save Translations

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple", Description: "Fresh fruit"},
    {ID: 2, Locale: gotrans.LocaleEN, Title: "Banana", Description: "Yellow fruit"},
}
err := translator.SaveTranslations(ctx, products)
```

### 4. Load Translations

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN},
    {ID: 2, Locale: gotrans.LocaleEN},
}
products, err := translator.LoadTranslations(ctx, products)
fmt.Printf("Product 1: %s - %s\n", products[0].Title, products[0].Description)
// Output: Product 1: Apple - Fresh fruit
```

### 5. Delete Translations

```go
// Delete specific fields for specific locale
err := translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", 
    []int{1, 2}, []string{"title", "description"})

// Delete all translations for entities (all locales)
err := translator.DeleteTranslationsByEntity(ctx, "product", []int{1, 2})
```

## How It Works

### Translatable Interface

Every translatable entity must implement three methods:

```go
type Translatable interface {
    // TranslationLocale returns the language for this entity
    TranslationLocale() gotrans.Locale
    
    // TranslationEntityID returns the unique identifier
    TranslationEntityID() int
    
    // TranslatableFields returns struct field to database field mapping
    // Key: struct field name (e.g., "Title")
    // Value: database field ID (e.g., "title")
    TranslatableFields() map[string]string
}
```

The field mapping separates struct naming (PascalCase) from database naming conventions:

```go
func (p Product) TranslatableFields() map[string]string {
    return map[string]string{
        "Title":            "title",              // struct field -> DB field
        "Description":     "description",
        "AIRecommendation": "ai_recommendation",  // map to any db column name
    }
}
```

### Translator Interface

```go
type Translator[T Translatable] interface {
    // LoadTranslations fetches translations and populates string fields
    LoadTranslations(ctx context.Context, entities []T) ([]T, error)
    
    // SaveTranslations persists translations (creates or updates)
    SaveTranslations(ctx context.Context, entities []T) error
    
    // DeleteTranslations removes specific translations
    DeleteTranslations(ctx context.Context, locale Locale, entity string, 
        entityIDs []int, fields []string) error
    
    // DeleteTranslationsByEntity removes all translations for entities
    DeleteTranslationsByEntity(ctx context.Context, entity string, 
        entityIDs []int) error
}
```

## Automatic Optimization

When you work with multiple locales, the translator automatically groups translations by locale for efficient batch operations:

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple"},
    {ID: 1, Locale: gotrans.LocaleFR, Title: "Pomme"},
    {ID: 2, Locale: gotrans.LocaleEN, Title: "Banana"},
    {ID: 2, Locale: gotrans.LocaleFR, Title: "Banane"},
}

// Automatically makes 2 DB calls (grouped by locale)
// Instead of 4 individual calls
translator.SaveTranslations(ctx, products)
```

### Performance Metrics

| Scenario | Database Calls | Improvement |
|----------|---|---|
| 100 entities, 1 locale | 1 | 100x faster |
| 100 entities, 2 locales | 2 | 50x faster |
| 100 entities, 5 locales | 5 | 20x faster |
| 1000 entities, 10 locales | 10 | 100x faster |

## Database Schema

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

**Fields:**
- `entity`: Entity type name (auto-converted to snake_case from struct name)
- `entity_id`: Entity's primary key
- `field`: Translatable field ID (from your mapping)
- `locale`: ISO-639-1 language code
- `value`: Translated text

**Unique Constraint**: Ensures no duplicate translations for the same entity, field, and locale.

## Entity Name Resolution

Entity names are automatically converted from struct names to snake_case:

```
Product          → product
ProductCategory  → product_category
OrderItem        → order_item
AIRecommendation → ai_recommendation
```

This conversion is automatic and handled internally.

## Supported Locales

The library includes 41 language locales:

```go
gotrans.LocaleEN    // English
gotrans.LocaleFR    // French
gotrans.LocaleDE    // German
gotrans.LocaleES    // Spanish
gotrans.LocaleIT    // Italian
gotrans.LocaleRU    // Russian
gotrans.LocaleJA    // Japanese
gotrans.LocaleZH    // Chinese
gotrans.LocaleKO    // Korean
gotrans.LocaleAR    // Arabic
// ... and 31 more
```

Use `gotrans.ParseLocale(code)` to convert string codes to Locale constants:

```go
locale, ok := gotrans.ParseLocale("en")
if ok {
    fmt.Println(locale == gotrans.LocaleEN) // true
}
```

## Multi-Locale Operations

Handle multiple languages in a single operation:

```go
// Load English version
productsEN := []Product{
    {ID: 1, Locale: gotrans.LocaleEN},
    {ID: 2, Locale: gotrans.LocaleEN},
}
productsEN, _ = translator.LoadTranslations(ctx, productsEN)

// Load French version
productsFR := []Product{
    {ID: 1, Locale: gotrans.LocaleFR},
    {ID: 2, Locale: gotrans.LocaleFR},
}
productsFR, _ = translator.LoadTranslations(ctx, productsFR)
```

Or load mixed locales in one call:

```go
mixed := []Product{
    {ID: 1, Locale: gotrans.LocaleEN},
    {ID: 1, Locale: gotrans.LocaleFR},
    {ID: 2, Locale: gotrans.LocaleEN},
}
mixed, _ := translator.LoadTranslations(ctx, mixed)
// Automatically optimized: 2 queries instead of 3
```

## Example Application

Run a complete working example with SQLite:

```bash
go run ./example/main.go
```

The example demonstrates:
- Creating tables in SQLite
- Saving translations for multiple locales
- Loading translations
- Deleting translations
- Multi-locale optimization in action

## Testing

Run all tests:

```bash
go test -v ./...
```

All tests pass including:
- Single locale operations
- Multi-locale operations
- Deletion operations
- Locale parsing

## Use Cases

### E-commerce Platforms
```go
type Product struct {
    ID          int
    Locale      gotrans.Locale
    Name        string
    Description string
    Details     string
}
```

### CMS/Blog Systems
```go
type Article struct {
    ID       int
    Locale   gotrans.Locale
    Title    string
    Content  string
    Excerpt  string
}
```

### SaaS Applications
```go
type Feature struct {
    ID          int
    Locale      gotrans.Locale
    Name        string
    Description string
}
```

## Best Practices

1. **Always Set Locale** - Ensure every entity has the correct locale before save/load operations
2. **Use Field Mapping Consistently** - Keep mapping aligned between struct fields and database
3. **Leverage Batch Operations** - Pass multiple entities to save/load for better performance
4. **Handle Missing Translations** - Check if fields are empty after loading
5. **Use Transactions** - Wrap multiple save operations in database transactions for consistency

## Implementation Details

### Reflection Usage

The library uses reflection only where necessary:
- During `SaveTranslations()`: To extract string field values
- During `LoadTranslations()`: To apply fetched translations to struct fields

This is acceptable because:
- These operations are not in hot loops (typically called per request/batch)
- Performance impact is negligible compared to database I/O
- It provides flexibility and type safety

### Type Safety

The library uses Go generics to ensure compile-time type checking:

```go
translator := gotrans.NewTranslator[Product](repo)
// Only Product entities can be used with this translator
// Compile-time error if you try to use other types
```

## Related Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Deep dive into design and optimization
- **[QUICK_START.md](QUICK_START.md)** - Quick reference card
- **[FAQ.md](FAQ.md)** - Common questions and answers
- **[INDEX.md](INDEX.md)** - Complete documentation index

## License

MIT License - See LICENSE file for details

