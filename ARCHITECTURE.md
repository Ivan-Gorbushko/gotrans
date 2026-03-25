# Architecture & Design

## Overview

`gotrans` is a translation library that embeds locale information directly within entities. This design enables automatic optimization and a cleaner API compared to passing locale as a separate parameter.

## Core Design Principles

### 1. Embedded Locale

Each entity carries its own locale information via the `TranslationLocale()` method. This means:

```go
type Product struct {
    ID     int
    Locale gotrans.Locale  // ← Embedded locale
    Title  string
}

func (p Product) TranslationLocale() gotrans.Locale {
    return p.Locale
}
```

**Benefits:**
- Locale is part of the entity's semantics
- No need for separate function parameters
- Enables automatic grouping optimization
- Type-safe design

### 2. Explicit Field Mapping

The `TranslatableFields()` method provides explicit association between struct fields and database field IDs:

```go
func (p Product) TranslatableFields() map[string]string {
    return map[string]string{
        "Title":       "title",
        "Description": "description",
    }
}
```

**Benefits:**
- Struct field names are decoupled from database names
- PascalCase struct fields can map to snake_case database fields
- Easy to rename fields without breaking translations
- Clear, explicit mapping prevents bugs

### 3. Automatic Locale Grouping

When saving or loading translations, the library automatically groups by locale:

```go
// Developer writes:
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple"},
    {ID: 1, Locale: gotrans.LocaleFR, Title: "Pomme"},
    {ID: 2, Locale: gotrans.LocaleEN, Title: "Banana"},
}
translator.SaveTranslations(ctx, products)

// Internally:
// Group by locale:
//   EN: [prod1, prod2]
//   FR: [prod1]
// Save EN group -> 1 DB call
// Save FR group -> 1 DB call
// Total: 2 DB calls instead of 3
```

## Optimization Strategy

### Save Operations

```go
func (t *translator[T]) SaveTranslations(ctx context.Context, entities []T) error {
    // Step 1: Group translations by locale
    localeMap := make(map[Locale][]Translation)
    for _, e := range entities {
        locale := e.TranslationLocale()
        translations := extractTranslations(...)
        localeMap[locale] = append(localeMap[locale], translations...)
    }
    
    // Step 2: Save each locale group
    for locale, translations := range localeMap {
        t.translationRepository.MassCreateOrUpdate(ctx, locale, translations)
    }
    return nil
}
```

**Performance Improvement:**
- Single locale: 1x (same as before)
- Multiple locales: N to 1 ratio (N = number of entities)
- Example: 100 entities × 1 locale = 1 call (not 100)

### Load Operations

```go
func (t *translator[T]) LoadTranslations(ctx context.Context, entities []T) ([]T, error) {
    // Step 1: Group entities by locale
    localeMap := make(map[Locale][]int)
    for _, e := range entities {
        locale := e.TranslationLocale()
        localeMap[locale] = append(localeMap[locale], e.TranslationEntityID())
    }
    
    // Step 2: Load translations for each locale group
    for locale, entityIDs := range localeMap {
        translations := t.translationRepository.GetTranslations(ctx, locale, entityType, entityIDs)
        allTranslations = append(allTranslations, translations...)
    }
    
    // Step 3: Apply to entities
    for i := range entities {
        t.applyTranslations(&entities[i], allTranslations)
    }
    return entities, nil
}
```

## Database Design

### Schema

```sql
CREATE TABLE translations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    entity VARCHAR(100) NOT NULL,
    entity_id BIGINT NOT NULL,
    field VARCHAR(100) NOT NULL,
    locale VARCHAR(10) NOT NULL,
    value TEXT NOT NULL,
    UNIQUE(entity, entity_id, field, locale)
);
```

### Why This Design?

1. **Normalized**: Each translation is a separate row
2. **Scalable**: Works with any number of locales
3. **Flexible**: Field mapping handled in code, not database
4. **Consistent**: Unique constraint prevents duplicates

### Entity Name Resolution

Entity type names are converted to snake_case:

```
Product          → product
ProductCategory  → product_category
OrderItem        → order_item
```

This is handled by the `toSnakeCase()` utility function with special handling for acronyms:

```
AIRecommends  → ai_recommends
AICRecommends → aic_recommends
```

## Translator Interface

### LoadTranslations

```go
LoadTranslations(ctx context.Context, entities []T) ([]T, error)
```

- Groups entities by locale
- Fetches translations for each locale
- Populates string fields from translations
- Returns modified entities

**Key Point**: Only string fields are populated. Non-string fields remain unchanged.

### SaveTranslations

```go
SaveTranslations(ctx context.Context, entities []T) error
```

- Extracts translatable field values
- Groups by locale
- Uses `MassCreateOrUpdate` for each locale
- Handles creation and updates automatically

### DeleteTranslations

```go
DeleteTranslations(ctx context.Context, locale Locale, entity string, 
    entityIDs []int, fields []string) error
```

- Deletes specific translations
- Still requires locale parameter (for specificity)
- Can target specific fields or all fields

### DeleteTranslationsByEntity

```go
DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error
```

- Deletes all translations for entities
- Across all locales
- Useful for entity deletion cleanup

## Reflection Usage

The library uses reflection in two places:

### 1. SaveTranslations (extractTranslations function)

```go
func extractTranslations(entityName string, entityID int, entity any, locale Locale) ([]Translation, error) {
    v := reflect.ValueOf(entity)
    fieldMap := translatable.TranslatableFields()
    
    // For each field in field map
    for name, fieldID := range fieldMap {
        // Get struct field value
        field := v.FieldByName(name)
        if field.Kind() == reflect.String {
            // Add translation
            results = append(results, Translation{...})
        }
    }
    return results, nil
}
```

### 2. LoadTranslations (applyTranslations function)

```go
func (t *translator[T]) applyTranslations(entity *T, translations []Translation) error {
    v := reflect.ValueOf(entity).Elem()
    fieldMap := translatable.TranslatableFields()
    
    // For each translation matching this entity
    for _, tr := range translations {
        if idx, ok := idToIndex[tr.Field]; ok {
            field := v.Field(idx)
            if field.Kind() == reflect.String && field.CanSet() {
                field.SetString(tr.Value)
            }
        }
    }
    return nil
}
```

**Why Reflection?**
- Provides flexibility (works with any struct)
- Eliminates boilerplate code generation
- Performance impact negligible (only during save/load, not per-query)

## Type Safety

### Generics

The library uses Go 1.18+ generics for type safety:

```go
translator := gotrans.NewTranslator[Product](repo)
```

**Benefits:**
- Compile-time type checking
- No type assertions or casting needed
- IDE autocomplete works perfectly
- Impossible to mix entity types

### Interface Compliance

Compile-time verification that entity implements `Translatable`:

```go
var _ Translatable = (*Product)(nil)
```

## Implementation Quality

### Error Handling

All operations return error values:

```go
err := translator.SaveTranslations(ctx, products)
if err != nil {
    // Handle database or validation errors
}
```

### Context Support

All operations accept `context.Context`:

```go
translator.LoadTranslations(ctx, products)
```

Enables:
- Timeout control
- Cancellation
- Request-scoped values

### Concurrency

The library is safe for concurrent use:
- Translator is stateless
- Repository operations are database-level atomic
- No shared mutable state

## Performance Characteristics

### Best Case
- Single entity, single locale: 2 queries (1 select, 1 upsert)

### Common Case
- 100 entities, single locale: 2 queries (1 batch select, 1 batch upsert)

### Good Case
- 100 entities, mixed locales: 2N queries where N = number of locales

## Limitations

### String Fields Only

Only `string` type fields are translatable. Other types are skipped:

```go
type Product struct {
    Price   float64 // Not translatable
    Title   string  // Translatable ✓
    InStock bool    // Not translatable
}
```

### No Nested Objects

Translation applies to top-level struct fields only:

```go
type Product struct {
    Title   string // Works ✓
    Details struct {
        Description string // Not translatable
    }
}
```

### Explicit Mapping Required

Field mapping is required and must match actual fields:

```go
func (p Product) TranslatableFields() map[string]string {
    return map[string]string{
        "Title": "title", // Must match actual field name
    }
}
```

## Database Compatibility

Tested with:
- MySQL 5.7+
- MySQL 8.0+
- SQLite 3.x
- PostgreSQL (via sqlx compatibility)

Uses sqlx for database abstraction, so compatible with any sqlx-supported database.

## Supported Locales

41 language locales with ISO-639-1 codes:

- English, French, German, Spanish, Italian, Portuguese
- Russian, Ukrainian, Polish, Czech, Slovak, Hungarian
- Chinese, Japanese, Korean, Vietnamese, Thai, Indonesian
- Arabic, Hebrew, Turkish, Persian
- Bulgarian, Croatian, Serbian, Slovenian, Romanian
- Lithuanian, Latvian, Norwegian, Swedish, Danish, Finnish, Estonian
- Georgian, Kazakh, Macedonian, Albanian, Bosnian, Azerbaijani

Access via constants: `gotrans.LocaleEN`, `gotrans.LocaleFR`, etc.

Convert from strings: `gotrans.ParseLocale("en")`

