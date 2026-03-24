# FAQ - Frequently Asked Questions

## General Questions

### Q: Why embed locale in the entity instead of passing it as a parameter?

**A:** There are several reasons:

1. **Cleaner API**: Fewer function parameters make the code more readable
2. **Type Safety**: Locale becomes part of the entity contract
3. **Performance**: Enables automatic grouping optimization
4. **Semantics**: It makes sense logically - an entity in a specific locale

### Q: Does this affect database performance?

**A:** Actually, it **improves** database performance:

- **Save Operations**: If you save 100 entities with mixed locales, the old API would make 100 database calls (one per entity). The new API groups by locale and makes fewer calls.
- **Example**: 100 entities with 2 locales = 2 database calls instead of 100
- **Load Operations**: No change - already optimized

### Q: How do I migrate from the old API?

**A:** See the "Migration Path" section in `ARCHITECTURE.md` for detailed examples.

## Technical Questions

### Q: What if my entity doesn't need translations?

**A:** Then you don't need the translator. Just use a normal entity without implementing `Translatable`.

### Q: Can I translate different entities with different locales in a single call?

**A:** Yes! That's actually the strong point of this design:

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple"},
    {ID: 2, Locale: gotrans.LocaleFR, Title: "Pomme"},
}
translator.SaveTranslations(ctx, products)
```

The translator will automatically group them by locale and save efficiently.

### Q: Can I change the locale of an entity after creation?

**A:** Yes, you can change the `Locale` field anytime:

```go
product.Locale = gotrans.LocaleFR
translator.LoadTranslations(ctx, []Product{product})
```

This will load French translations for that product.

### Q: What happens if the locale is LocaleNone?

**A:** `LocaleNone` (value 0) is used for special cases like deleting all translations for an entity regardless of locale. For normal translation operations, always set a specific locale.

### Q: Can I have multiple translators?

**A:** Yes, you can create multiple translator instances:

```go
productTranslator := gotrans.NewTranslator[Product](repo)
categoryTranslator := gotrans.NewTranslator[Category](repo)
```

Both use the same repository, but work with different entity types.

## Field Mapping Questions

### Q: Why do I need TranslatableFieldMap()?

**A:** Because struct field names might not match database field IDs:

- Struct field: `Title` (PascalCase)
- DB field: `title` (lowercase)
- DB field: `product_title` (with prefix)

The map explicitly defines the relationship.

### Q: What field types are supported?

**A:** Currently only `string` fields are translatable. If you try to translate other types, they're skipped.

```go
type Product struct {
    ID    int     // Not translatable
    Title string  // Translatable ✓
    Price float64 // Not translatable
}
```

### Q: Can I translate computed fields?

**A:** No, only struct fields. If you need to translate computed values, store them as regular fields first.

## Database Questions

### Q: What database systems are supported?

**A:** Any database supported by `sqlx`:
- MySQL
- PostgreSQL
- SQLite
- Oracle
- SQL Server
- And others

### Q: Can I use a different table name?

**A:** Yes, modify the MySQL repository implementation to use a different table name. The current implementation hardcodes `translations`, but you can fork and customize.

### Q: What's the unique key for?

**A:** The unique constraint ensures no duplicate translations:

```sql
UNIQUE KEY uniq_translation (entity, entity_id, field, locale)
```

This prevents having two different values for the same field in the same language.

### Q: Can I add custom columns to the translations table?

**A:** The library only uses the columns it knows about. You can add custom columns, but they won't be used by the translator.

## Performance Questions

### Q: How much faster is batch save with grouping?

**A:** Depends on the number of locales:

- 100 entities, 1 locale: 100x faster (100 calls → 1 call)
- 100 entities, 2 locales: 50x faster (100 calls → 2 calls)
- 100 entities, 10 locales: 10x faster (100 calls → 10 calls)

The improvement is proportional to the number of entities divided by the number of locales.

### Q: Does reflection impact performance?

**A:** Reflection is only used during `SaveTranslations()` and `LoadTranslations()`, not in query paths. The impact is negligible compared to database I/O.

### Q: Can I pre-allocate slices for better performance?

**A:** Yes, if you know the expected number of translations:

```go
translations := make([]Translation, 0, expectedSize)
```

The library will still work with dynamically allocated slices.

## Testing Questions

### Q: How do I test my translatable entities?

**A:** Mock the repository:

```go
mockRepo := &mockRepo{
    translations: []gotrans.Translation{
        // Your test data
    },
}
translator := gotrans.NewTranslator[MyEntity](mockRepo)
```

See `gotrans_test.go` for examples.

### Q: Do I need to test the translator itself?

**A:** No, it's already well-tested. Focus on testing your entity implementation and business logic.

## Troubleshooting

### Q: I'm getting "not all code paths return a value"

**A:** Make sure you're implementing all three interface methods:
- `TranslationLocale()` - returns Locale
- `TranslationEntityID()` - returns int
- `TranslatableFieldMap()` - returns map[string]string

### Q: Translations aren't being loaded

**A:** Check:
1. Is `TranslatableFieldMap()` returning the correct field IDs?
2. Is the entity locale set correctly before loading?
3. Do translations exist in the database for that locale and entity ID?

### Q: Translations aren't being saved

**A:** Check:
1. Are the translatable fields (string fields) populated?
2. Is the entity locale set before saving?
3. Is the repository connected to a working database?

### Q: I'm getting unique constraint violations

**A:** This happens when trying to insert duplicate translations. The library uses `MassCreateOrUpdate` which deletes before inserting. If you're calling save multiple times in quick succession, you might hit a race condition. Use transactions to be safe.

## Feature Requests

### Q: Can I have nested/hierarchical translations?

**A:** Not directly. You would need to flatten them into separate string fields.

### Q: Can I translate non-string fields?

**A:** Not with the current implementation. You'd need to modify the `extractTranslations()` and `applyTranslations()` functions to support other types.

### Q: Can I have partial translations (missing some locales)?

**A:** Yes, the library handles this gracefully:
- If a translation doesn't exist for a locale, the field remains as-is
- You can check if a field is empty to see if translation is missing

## Related Resources

- See `ARCHITECTURE.md` for design decision details
- See `example/main.go` for working examples
- See `gotrans_test.go` for test examples
- See `README.md` for API documentation

