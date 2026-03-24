# gotrans
This repository provides a lightweight, framework-agnostic translation module for Golang applications. It is designed to manage multi-language content directly within backend business logic, without relying on heavy external localization frameworks.

## Concept
- Translatable fields in your entities are plain string fields (not map or struct).
- Each entity carries its own locale information via the `TranslationLocale()` method.
- You explicitly associate struct fields with translation field IDs via the `Translatable` interface.
- The translator automatically groups translations by locale for optimized database operations.
- The repository stores translations in a normalized table (see below).

## Entity Example
```go
// Example entity with built-in locale
type Product struct {
    ID          int
    Locale      gotrans.Locale  // Built-in locale field
    Title       string
    Description string
}

// Implement Translatable interface
func (p Product) TranslationLocale() gotrans.Locale { return p.Locale }
func (p Product) TranslationEntityID() int          { return p.ID }
func (p Product) TranslatableFieldMap() map[string]string {
    return map[string]string{
        "Title":       "title",       // Struct field -> DB field mapping
        "Description": "description",
    }
}
```

## Usage Example
```go
ctx := context.Background()
repo := mysql.NewTranslationRepository(db) // db: *sqlx.DB (MySQL or SQLite)
translator := gotrans.NewTranslator[Product](repo)

// Save translations for English locale
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple", Description: "Fresh fruit"},
    {ID: 2, Locale: gotrans.LocaleEN, Title: "Banana", Description: "Yellow fruit"},
}
_ = translator.SaveTranslations(ctx, products)

// Load translations for English locale
products = []Product{
    {ID: 1, Locale: gotrans.LocaleEN},
    {ID: 2, Locale: gotrans.LocaleEN},
}
products, _ := translator.LoadTranslations(ctx, products)
fmt.Printf("Product 1: %s - %s\n", products[0].Title, products[0].Description)

// Delete translations
_ = translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", []int{1, 2}, []string{"title", "description"})

// Delete all translations for entities
_ = translator.DeleteTranslationsByEntity(ctx, "product", []int{1, 2})
```

## Translatable Interface
```go
type Translatable interface {
    // Returns the locale for this entity's translations
    TranslationLocale() gotrans.Locale
    
    // Returns the entity ID (primary key)
    TranslationEntityID() int
    
    // Returns mapping of struct field names to translation field IDs
    // Example: map[string]string{"Title": "title", "Description": "description"}
    TranslatableFieldMap() map[string]string
}
```


## MySQL Table Structure
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
- **Optimized Batch Operations**: Automatically groups translations by locale for efficient database operations
- **Built-in Locale Support**: Entities carry their own locale information - no need for separate locale parameters
- **Framework Agnostic**: Works with any SQL database (MySQL, SQLite, PostgreSQL, etc.) via sqlx
- **Type Safe**: Uses Go generics for type-safe translation operations
- **Explicit Field Mapping**: Clear mapping between struct fields and translation field IDs prevents mistakes
- **Minimal Reflection**: Only uses reflection where necessary (during load/save operations)
- **Easy Integration**: Simple API that integrates seamlessly with any Go project

## How It Works

### Saving Translations
When you call `SaveTranslations(ctx, entities)`, the translator:
1. Extracts the locale from each entity via `TranslationLocale()`
2. Groups translations by locale for batch processing
3. Calls `MassCreateOrUpdate()` once per locale group
4. Reduces database round-trips and improves performance

### Loading Translations
When you call `LoadTranslations(ctx, entities)`, the translator:
1. Groups entities by their locale via `TranslationLocale()`
2. Fetches translations for each locale group in parallel groups
3. Applies translations to each entity using the field mapping
4. Returns entities with translated string fields populated

## Example with SQLite
Run the complete example with SQLite:
```bash
go run ./example/main.go
```

This demo shows how to create tables, save, load, and delete translations.

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Detailed design decisions and how the optimization works
- **[REFACTOR.md](REFACTOR.md)** - Summary of changes from the previous version
- **[FAQ.md](FAQ.md)** - Common questions and answers

## Supported Locales

The library includes support for 41 language locales:
- English, French, German, Spanish, Italian, Portuguese
- Russian, Ukrainian, Polish, Czech, Slovak, Hungarian
- Chinese, Japanese, Korean, Vietnamese, Thai, Indonesian
- Arabic, Hebrew, Turkish, Persian
- Bulgarian, Croatian, Serbian, Slovenian, Romanian, Lithuanian, Latvian
- Norwegian, Swedish, Danish, Finnish, Estonian
- Georgian, Kazakh, Macedonian, Albanian, Bosnian, Azerbaijani

Access them via constants: `gotrans.LocaleEN`, `gotrans.LocaleFR`, etc.

Use `gotrans.ParseLocale(string)` to convert string codes to Locale constants.

## Best Practices

1. **Always set Locale before operations**: Ensure each entity has the correct locale before load/save
2. **Use field mapping carefully**: Keep the mapping consistent between struct fields and database field IDs
3. **Batch operations**: Leverage automatic grouping by passing multiple entities with different locales
4. **Error handling**: Always check the error return value
5. **Transactions**: Use database transactions for consistency when saving multiple batches

## License

MIT License - See LICENSE file for details

