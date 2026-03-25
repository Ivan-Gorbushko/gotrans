# FAQ - Frequently Asked Questions

## General Questions

### Q: Why embed locale in the entity?

**A:** Entities are self-describing and carry all context needed for translation operations. This enables automatic grouping optimization and eliminates redundant parameters.

### Q: Does this affect database performance?

**A:** It improves it. The library automatically groups translations by locale before saving, reducing database calls significantly (up to 100x for batch operations).

### Q: What if my entity doesn't need translations?

**A:** Then you don't use the library. Just use plain structs without implementing the `Translatable` interface.

## Technical Questions

### Q: Can I translate different entity types together?

**A:** Each translator is specific to one entity type via generics. Create multiple translators for different types:

```go
productTrans := gotrans.NewTranslator[Product](repo)
categoryTrans := gotrans.NewTranslator[Category](repo)
```

### Q: Can I use mixed locales in one call?

**A:** Yes, that's actually the strong point:

```go
entities := []Product{
    {ID: 1, locale: gotrans.LocaleEN, Title: "Apple"},
    {ID: 2, locale: gotrans.LocaleFR, Title: "Pomme"},
}
translator.SaveTranslations(ctx, entities)
```

The library automatically optimizes this.

### Q: Can I change locale after creation?

**A:** Yes:

```go
product.locale = gotrans.LocaleFR  // Note: private field, set via struct initialization
products, _ := translator.LoadTranslations(ctx, []Product{product})
```

### Q: What is LocaleNone?

**A:** It's the zero value (0). Use it only in `DeleteTranslationsByEntity` to delete across all locales.

## Field Mapping Questions

### Q: Why do I need TranslatableFields()?

**A:** It decouples struct field names from database field names:

```go
// Struct: Title (PascalCase)
// Database: title (snake_case)
return map[string]string{
    "Title": "title",  // ← Explicit mapping
}
```

### Q: What field types are supported?

**A:** Only `string` fields are translatable. Other types are ignored.

```go
type Product struct {
    ID    int     // ✗ Not translatable
    Title string  // ✓ Translatable
    Price float64 // ✗ Not translatable
}
```

### Q: Can I translate computed/derived fields?

**A:** No. Only struct fields. Store computed values as regular fields first.

## Database Questions

### Q: What databases are supported?

**A:** Any database supported by sqlx:
- MySQL
- PostgreSQL
- SQLite
- Oracle
- SQL Server

### Q: Can I use a different table name?

**A:** The repository hardcodes `translations`. To use a different name, implement a custom repository.

### Q: What's the unique constraint for?

**A:** It prevents duplicate translations:

```sql
UNIQUE(entity, entity_id, field, locale)
```

One value per entity, field, and locale.

### Q: Can I add custom columns?

**A:** You can, but the library won't use them. It only works with the defined columns.

## Performance Questions

### Q: How much faster is batch save?

**A:**
- 100 entities, 1 locale: **100x faster** (100 calls → 1)
- 100 entities, 2 locales: **50x faster** (100 calls → 2)
- 100 entities, 10 locales: **10x faster** (100 calls → 10)

The improvement depends on the locale distribution.

### Q: Does reflection impact performance?

**A:** No. Reflection is only used during save/load, not in query paths. Database I/O dominates.

### Q: Can I pre-allocate slices?

**A:** Yes, for better memory efficiency, but it's not required. The library handles dynamic allocation.

## Testing Questions

### Q: How do I test with this library?

**A:** Mock the repository:

```go
mockRepo := &mockRepository{
    data: []gotrans.Translation{ /* test data */ },
}
translator := gotrans.NewTranslator[MyEntity](mockRepo)
```

See `gotrans_test.go` for examples.

### Q: Do I need to test the library itself?

**A:** No. Focus on testing your entity implementation and business logic. The library is well-tested.

## Troubleshooting

### Issue: "does not satisfy Translatable"

**Solution**: Implement all required methods:

```go
func (p Product) TranslationLocale() gotrans.Locale { return p.locale }
func (p Product) TranslationEntityID() int { return p.ID }
func (p Product) TranslatableFields() map[string]string { /* ... */ }
func (p Product) TranslationEntityName() string { return "product" }
```

### Issue: Translations not loading

**Check**:
1. Is `TranslatableFields()` mapping correct?
2. Is entity locale set before load?
3. Do translations exist in the database?
4. Does `TranslationEntityName()` match the database entity name?

```go
product := Product{ID: 1, locale: gotrans.LocaleEN}
products, _ := translator.LoadTranslations(ctx, []Product{product})
```

### Issue: Unique constraint violation

**Cause**: You're inserting duplicate translations.

**Solution**: `SaveTranslations` uses `MassCreateOrUpdate` which handles this. Use it consistently.

### Issue: Empty fields after load

**Cause**: Translations don't exist for that locale.

**Solution**: Check if the field is empty:

```go
if product.Title == "" {
    fmt.Println("Translation not found")
}
```

## Supported Locales

The library includes 41 languages:

**European**: English, French, German, Spanish, Italian, Portuguese, Russian, Ukrainian, Polish, Czech, Slovak, Hungarian, Bulgarian, Croatian, Serbian, Slovenian, Romanian, Lithuanian, Latvian, Norwegian, Swedish, Danish, Finnish, Estonian

**Asian**: Chinese, Japanese, Korean, Vietnamese, Thai, Indonesian

**Middle Eastern/African**: Arabic, Hebrew, Turkish, Persian, Georgian, Kazakh, Macedonian, Albanian, Bosnian, Azerbaijani

Use constants: `gotrans.LocaleEN`, `gotrans.LocaleFR`, etc.

Convert from codes: `gotrans.ParseLocale("en")`

## Migration Guide

If you're integrating this library into an existing system:

### Step 1: Add Locale Field

```go
type MyEntity struct {
    ID     int
    locale gotrans.Locale  // ← Add this (private field)
    // ... other fields
}
```

### Step 2: Implement Interface

```go
func (m MyEntity) TranslationLocale() gotrans.Locale { return m.locale }
func (m MyEntity) TranslationEntityID() int { return m.ID }
func (m MyEntity) TranslatableFields() map[string]string {
    return map[string]string{
        "Field1": "field_1",
        "Field2": "field_2",
    }
}
func (m MyEntity) TranslationEntityName() string { return "my_entity" }
```

### Step 3: Create Translator

```go
repo := mysql.NewTranslationRepository(db)
translator := gotrans.NewTranslator[MyEntity](repo)
```

### Step 4: Use API

```go
// Save
entity := MyEntity{ID: 1, locale: gotrans.LocaleEN}
translator.SaveTranslations(ctx, []MyEntity{entity})

// Load
entity = MyEntity{ID: 1, locale: gotrans.LocaleEN}
entities, _ := translator.LoadTranslations(ctx, []MyEntity{entity})

// Delete
translator.DeleteTranslations(ctx, gotrans.LocaleEN, "my_entity", []int{1}, []string{"field_1"})
```

## Feature Requests

### Q: Can I have nested/hierarchical translations?

**A:** Not directly. Flatten them into separate string fields.

### Q: Can I translate non-string types?

**A:** Not built-in. You'd need to modify the library to support it.

### Q: Can I have partial translations?

**A:** Yes. If a translation is missing, the field remains as-is.

## Related Resources

- **README.md** - Quick start guide
- **ARCHITECTURE.md** - Design details
- **QUICK_START.md** - Quick reference
- **example/main.go** - Working examples

