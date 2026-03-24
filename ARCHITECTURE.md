# Architecture & Design Decisions

## Overview
This document explains the key architectural decisions made in the `gotrans` library refactor.

## Problem Statement
The original implementation required passing `locale` as a separate parameter to `LoadTranslations()` and `SaveTranslations()`. This approach had several issues:
1. Locale parameter had to be repeated across multiple method calls
2. No automatic optimization for batch operations with mixed locales
3. Unclear API where locale context wasn't embedded in entities

## Solution: Locale Inside Entity

### Key Changes

#### 1. Locale Embedded in Entity
Instead of passing locale as a parameter, each entity now carries its own locale:

```go
type Product struct {
    ID          int
    Locale      gotrans.Locale  // <-- Embedded locale
    Title       string
    Description string
}

func (p Product) TranslationLocale() gotrans.Locale { return p.Locale }
```

**Benefits:**
- Cleaner API: `translator.SaveTranslations(ctx, products)`
- Entities are self-describing (they know their own locale)
- Enables grouping optimization automatically

#### 2. Automatic Locale Grouping
The translator now automatically groups translations by locale before saving:

```go
func (t *translator[T]) SaveTranslations(ctx context.Context, entities []T) error {
    // Group translations by locale
    localeMap := make(map[Locale][]Translation)
    for _, e := range entities {
        locale := e.TranslationLocale()
        // Accumulate translations by locale
        localeMap[locale] = append(localeMap[locale], translations...)
    }
    
    // Save each locale group separately
    for locale, translations := range localeMap {
        if err := t.translationRepository.MassCreateOrUpdate(ctx, locale, translations); err != nil {
            return err
        }
    }
    return nil
}
```

**Performance Impact:**
- If you save 100 entities with mixed locales (50 EN, 50 FR), the system makes 2 database calls instead of 100
- Each call is optimized for a single locale batch
- Significantly reduces database overhead for multi-locale operations

#### 3. Explicit Field Mapping
The `TranslatableFieldMap()` method provides explicit association between struct fields and translation field IDs:

```go
func (p Product) TranslatableFieldMap() map[string]string {
    return map[string]string{
        "Title":       "title",        // Struct field → DB field
        "Description": "description",
    }
}
```

**Benefits:**
- Type-safe: Field names are checked at compile time
- Decouples: Struct field names from database field IDs
- Flexible: Can rename fields without breaking translations
- Clear: Explicit mapping prevents confusion

## Database Schema

The translations are stored in a single normalized table:

```sql
CREATE TABLE translations (
    id BIGINT AUTO_INCREMENT,
    entity VARCHAR(100) NOT NULL,
    entity_id BIGINT NOT NULL,
    field VARCHAR(100) NOT NULL,
    locale VARCHAR(10) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uniq_translation (entity, entity_id, field, locale)
);
```

The unique constraint ensures no duplicate translations for the same entity, field, and locale.

## API Methods

### LoadTranslations
```go
func (t *translator[T]) LoadTranslations(ctx context.Context, entities []T) ([]T, error)
```
- Groups entities by locale
- Loads translations for each locale group
- Applies translations to struct fields using the field map
- Returns enriched entities

### SaveTranslations
```go
func (t *translator[T]) SaveTranslations(ctx context.Context, entities []T) error
```
- Extracts locale from each entity
- Groups translations by locale
- Calls `MassCreateOrUpdate()` once per locale
- Handles upsert semantics (create or update)

### DeleteTranslations
```go
func (t *translator[T]) DeleteTranslations(ctx context.Context, locale Locale, entity string, entityIDs []int, fields []string) error
```
- Deletes specific translation records
- Still requires explicit locale (used for querying)

### DeleteTranslationsByEntity
```go
func (t *translator[T]) DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error
```
- Deletes all translations for specified entities
- Deletes across all locales

## Reflection Usage

The library uses reflection only where necessary:
- **During load**: To extract translated values and set them on struct fields
- **During save**: To extract translatable field values from structs

This is acceptable because:
1. These operations are called relatively infrequently (not in hot loops)
2. The performance impact is minimal compared to database I/O
3. It provides type safety and flexibility

## Type Safety

The library uses Go generics to ensure type safety:

```go
translator := gotrans.NewTranslator[Product](repo)
```

This ensures:
- Only `Product` entities (or compatible types) can be passed
- Compile-time type checking
- No runtime type assertions needed

## Testing

The library includes comprehensive tests:
- `TestLoadTranslations`: Basic load functionality
- `TestSaveTranslations`: Basic save functionality
- `TestDeleteTranslations`: Deletion functionality
- `TestMultiLocaleSaveAndLoad`: Multi-locale optimization verification

## Example with SQLite

Run the example to see all features in action:

```bash
go run ./example/main.go
```

This demonstrates:
1. Creating tables
2. Saving translations in multiple locales
3. Loading translations
4. Deleting translations
5. Deleting all translations for entities

## Migration Path

If upgrading from the old API:

**Old API:**
```go
translator.SaveTranslations(ctx, gotrans.LocaleEN, products)
translator.LoadTranslations(ctx, gotrans.LocaleEN, products)
```

**New API:**
```go
// Add locale to your entity struct
type Product struct {
    ID     int
    Locale gotrans.Locale  // <-- Add this
    // ... other fields
}

// Implement TranslationLocale()
func (p Product) TranslationLocale() gotrans.Locale { return p.Locale }

// Update the entity with locale before operations
products[0].Locale = gotrans.LocaleEN

// Now use the cleaner API
translator.SaveTranslations(ctx, products)
translator.LoadTranslations(ctx, products)
```

