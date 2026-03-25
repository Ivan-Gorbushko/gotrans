# Architecture & Design

## Overview

`gotrans` is a translation library that embeds locale information directly within entities. This design enables automatic optimization and a cleaner API compared to passing locale as a separate parameter.

## Core Design Principles

### 1. Embedded Locale

Each entity carries its own locale information via the `TranslationLocale()` method. This means:

```go
type Product struct {
    ID     int
    locale gotrans.Locale  // ← Private field
    Title  string
}

func (p Product) TranslationLocale() gotrans.Locale {
    return p.locale
}
```

**Benefits:**
- Locale is part of the entity's semantics
- No need for separate function parameters
- Enables automatic grouping optimization
- Type-safe design
- Locale is private (not exported)

### 2. Explicit Entity Naming

The `TranslationEntityName()` method provides explicit control over entity naming:

```go
func (p Product) TranslationEntityName() string {
    return "product"  // Exact name as stored in DB
}
```

**Benefits:**
- Complete control over database naming
- Struct names can change without affecting translations
- Clear separation of concerns
- No automatic name conversion needed
- Optional: Use reflection helpers if you prefer automatic naming

### 3. Explicit Field Mapping

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

### 4. Automatic Locale Grouping

When saving or loading translations, the library automatically groups by locale:

```go
// Developer writes:
products := []Product{
    {ID: 1, locale: gotrans.LocaleEN, Title: "Apple"},
    {ID: 1, locale: gotrans.LocaleFR, Title: "Pomme"},
    {ID: 2, locale: gotrans.LocaleEN, Title: "Banana"},
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
    // Step 1: Get entity name from first entity
    entityName := entities[0].TranslationEntityName()
    
    // Step 2: Group translations by locale
    localeMap := make(map[Locale][]Translation)
    for _, e := range entities {
        locale := e.TranslationLocale()
        translations := extractTranslations(entityName, e.TranslationEntityID(), e, locale)
        localeMap[locale] = append(localeMap[locale], translations...)
    }
    
    // Step 3: Save each locale group
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
    // Step 1: Get entity name from first entity
    entityName := entities[0].TranslationEntityName()
    
    // Step 2: Group entities by locale
    localeMap := make(map[Locale][]int)
    for _, e := range entities {
        locale := e.TranslationLocale()
        localeMap[locale] = append(localeMap[locale], e.TranslationEntityID())
    }
    
    // Step 3: Load translations for each locale group
    var allTranslations []Translation
    for locale, entityIDs := range localeMap {
        translations := t.translationRepository.GetTranslations(ctx, locale, entityName, entityIDs)
        allTranslations = append(allTranslations, translations...)
    }
    
    // Step 4: Apply to entities
    for i := range entities {
        t.applyTranslations(&entities[i], allTranslations)
    }
    return entities, nil
}
```

## Database Design

### Schema

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

### Schema Explanation

- `id`: Auto-incrementing primary key
- `entity`: Entity name (from `TranslationEntityName()`)
- `entity_id`: Entity's primary key
- `field`: Translatable field ID (from `TranslatableFields()` mapping)
- `locale`: ISO-639-1 language code
- `value`: Translated text
- **Unique Constraint**: Prevents duplicate translations for the same entity, field, and locale

## Translator Interface

### LoadTranslations

```go
LoadTranslations(ctx context.Context, entities []T) ([]T, error)
```

- Gets entity name from first entity's `TranslationEntityName()` method
- Groups entities by locale
- Fetches translations for each locale group
- Populates string fields from translations
- Returns modified entities

**Key Point**: Only string fields are populated. Non-string fields remain unchanged.

### SaveTranslations

```go
SaveTranslations(ctx context.Context, entities []T) error
```

- Gets entity name from first entity's `TranslationEntityName()` method
- Extracts translatable field values using reflection
- Groups translations by locale
- Uses `MassCreateOrUpdate` for each locale group
- Handles creation and updates automatically

### DeleteTranslations

```go
DeleteTranslations(ctx context.Context, locale Locale, entity string, 
    entityIDs []int, fields []string) error
```

- Deletes specific translations
- Requires explicit locale parameter
- Can target specific fields or all fields

### DeleteTranslationsByEntity

```go
DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error
```

- Deletes all translations for entities
- Across all locales
- Useful for entity deletion cleanup

## Reflection Usage

The library uses reflection in three places:

### 1. SaveTranslations (extractTranslations function)

```go
func extractTranslations(entityName string, entityID int, entity any, locale Locale) ([]Translation, error) {
    v := reflect.ValueOf(entity)
    translatable := entity.(Translatable)
    fieldMap := translatable.TranslatableFields()
    
    // For each field in field map
    for name, fieldID := range fieldMap {
        // Get struct field value
        field := v.FieldByName(name)
        if field.Kind() == reflect.String {
            // Add translation
            results = append(results, NewTranslation(0, entityName, entityID, fieldID, locale, field.String()))
        }
    }
    return results, nil
}
```

**Purpose**: Extract string field values from struct for storage.

### 2. LoadTranslations (applyTranslations function)

```go
func (t *translator[T]) applyTranslations(entity *T, translations []Translation) error {
    v := reflect.ValueOf(entity).Elem()
    translatable := any(entity).(Translatable)
    entityName := translatable.TranslationEntityName()
    fieldMap := translatable.TranslatableFields()
    
    // For each translation matching this entity
    for _, tr := range translations {
        if tr.Entity == entityName && tr.GetLocale() == translatable.TranslationLocale() {
            if idx, ok := idToIndex[tr.Field]; ok {
                field := v.Field(idx)
                if field.Kind() == reflect.String && field.CanSet() {
                    field.SetString(tr.Value)
                }
            }
        }
    }
    return nil
}
```

**Purpose**: Apply fetched translations to struct fields.

### 3. GetEntityNameFromType / GetEntityNameFromValue (Helpers)

```go
func GetEntityNameFromType[T any](t *T) string {
    typ := reflect.TypeOf(t)
    if typ.Kind() == reflect.Ptr {
        typ = typ.Elem()
    }
    return toSnakeCase(typ.Name())
}
```

**Purpose**: Optional helper for automatic snake_case conversion from struct names.

**Why Reflection?**
- Provides flexibility (works with any struct)
- Eliminates boilerplate code generation
- Performance impact negligible (only during save/load, not in hot loops)
- Alternative: Implement `TranslationEntityName()` manually for full control

## Entity Naming Strategies

### Strategy 1: Manual Implementation (Recommended)

```go
type Product struct { ... }
func (p Product) TranslationEntityName() string { return "product" }

type GeoTag struct { ... }
func (g GeoTag) TranslationEntityName() string { return "geo_tag" }
```

**Pros:**
- Full control over naming
- Clear and explicit
- No hidden magic

**Cons:**
- Must implement for each type

### Strategy 2: Reflection Helper

```go
type Product struct { ... }
func (p Product) TranslationEntityName() string { return gotrans.GetEntityNameFromType(&p) }
```

**Pros:**
- Less boilerplate
- Automatic snake_case conversion

**Cons:**
- Less explicit
- Must remember to call helper

### Strategy 3: Static Function

```go
const ProductEntityName = "product"

type Product struct { ... }
func (p Product) TranslationEntityName() string { return ProductEntityName }
```

**Pros:**
- Reusable
- Single source of truth
- Easy to refactor

**Cons:**
- More code
- More constants to manage

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
var _ gotrans.Translatable = (*Product)(nil)
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

### Batch Operations

Example with 1000 entities, 10 locales:
- Without optimization: 1000 queries (1 per entity)
- With optimization: 20 queries (2 per locale)
- **Performance improvement: 50x faster**

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

Field mapping is required and must match actual struct fields:

```go
func (p Product) TranslatableFields() map[string]string {
    return map[string]string{
        "Title": "title", // Must match actual struct field name
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

## Summary

The architecture of `gotrans` is built on these principles:

1. **Simplicity**: Minimal API surface, easy to understand
2. **Safety**: Type-safe with generics, compile-time checking
3. **Performance**: Automatic optimization for batch operations
4. **Flexibility**: Explicit field mapping and entity naming
5. **Practicality**: Works with any sqlx-compatible database
6. **Maintainability**: Clear code, well-documented, tested

