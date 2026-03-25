package gotrans

import (
	"context"
	"reflect"
)

// Translatable is the interface every translatable entity must implement.
// TranslatableFields returns a map: struct field name → translation field ID in DB.
// Example: map[string]string{"Title": "title", "Description": "desc"}
// TranslationEntityName returns the name stored in the translations table.
type Translatable interface {
	TranslationEntityID() int
	TranslationEntityName() string
	TranslationEntityLocale() Locale
	TranslatableFields() map[string]string
}

// Translator is the main interface for translation operations.
// The entity name is derived from T at construction, so it does not need to be
// passed to the delete methods — the translator already knows it.
type Translator[T Translatable] interface {
	LoadTranslations(ctx context.Context, entities []T) ([]T, error)
	SaveTranslations(ctx context.Context, entities []T) error
	// DeleteTranslations removes translations for specific entity IDs, locale and fields.
	DeleteTranslations(ctx context.Context, locale Locale, entityIDs []int, fields []string) error
	// DeleteTranslationsByEntity removes all translations for the given entity IDs across all locales.
	DeleteTranslationsByEntity(ctx context.Context, entityIDs []int) error
}

var _ Translator[Translatable] = (*translator[Translatable])(nil)

type translator[T Translatable] struct {
	repo        TranslationRepository
	entityName  string          // derived from T once at construction, never changes
	fieldIndex  map[string]int  // DB field ID → struct field index, pre-built once
}

// NewTranslator creates a translator for entity type T.
// The entity name and field index are resolved once from a zero value of T.
func NewTranslator[T Translatable](repo TranslationRepository) Translator[T] {
	var zero T
	return &translator[T]{
		repo:       repo,
		entityName: zero.TranslationEntityName(),
		fieldIndex: buildFieldIndex[T](),
	}
}

func (t *translator[T]) DeleteTranslationsByEntity(ctx context.Context, entityIDs []int) error {
	return t.repo.MassDelete(ctx, LocaleNone, t.entityName, entityIDs, nil)
}

func (t *translator[T]) DeleteTranslations(ctx context.Context, locale Locale, entityIDs []int, fields []string) error {
	return t.repo.MassDelete(ctx, locale, t.entityName, entityIDs, fields)
}

func (t *translator[T]) LoadTranslations(ctx context.Context, entities []T) ([]T, error) {
	if len(entities) == 0 {
		return entities, nil
	}

	// Group entity IDs by locale for batch DB fetching.
	localeMap := make(map[Locale][]int)
	for _, e := range entities {
		locale := e.TranslationEntityLocale()
		localeMap[locale] = append(localeMap[locale], e.TranslationEntityID())
	}

	// Fetch translations for each locale group.
	var allTranslations []Translation
	for locale, ids := range localeMap {
		trs, err := t.repo.GetTranslations(ctx, locale, t.entityName, ids)
		if err != nil {
			return nil, err
		}
		allTranslations = append(allTranslations, trs...)
	}

	if len(allTranslations) == 0 {
		return entities, nil
	}

	// Build (entityID, locale) → []Translation lookup for O(1) access per entity.
	type key struct {
		id     int
		locale Locale
	}
	lookup := make(map[key][]Translation, len(entities))
	for _, tr := range allTranslations {
		k := key{tr.EntityID, tr.Locale}
		lookup[k] = append(lookup[k], tr)
	}

	// Apply translations to each entity using pre-built field index.
	for i := range entities {
		trs, ok := lookup[key{entities[i].TranslationEntityID(), entities[i].TranslationEntityLocale()}]
		if !ok {
			continue
		}
		v := reflect.ValueOf(&entities[i]).Elem()
		for _, tr := range trs {
			if idx, ok := t.fieldIndex[tr.Field]; ok {
				if f := v.Field(idx); f.Kind() == reflect.String && f.CanSet() {
					f.SetString(tr.Value)
				}
			}
		}
	}

	return entities, nil
}

func (t *translator[T]) SaveTranslations(ctx context.Context, entities []T) error {
	if len(entities) == 0 {
		return nil
	}

	// Group translations by locale for batch save.
	localeMap := make(map[Locale][]Translation)
	for _, e := range entities {
		trs := extractTranslations(e)
		locale := e.TranslationEntityLocale()
		localeMap[locale] = append(localeMap[locale], trs...)
	}

	for locale, trs := range localeMap {
		if len(trs) == 0 {
			continue
		}
		if err := t.repo.MassCreateOrUpdate(ctx, locale, trs); err != nil {
			return err
		}
	}

	return nil
}

// ------------------------------------------------
// --------------- Helpers ------------------------
// ------------------------------------------------

// buildFieldIndex builds a map from DB field ID → struct field index for type T.
// Called once per LoadTranslations since all entities share the same type.
func buildFieldIndex[T Translatable]() map[string]int {
	var zero T
	fieldMap := zero.TranslatableFields()
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	idToIndex := make(map[string]int, len(fieldMap))
	for i := 0; i < typ.NumField(); i++ {
		if fieldID, ok := fieldMap[typ.Field(i).Name]; ok {
			idToIndex[fieldID] = i
		}
	}
	return idToIndex
}

// extractTranslations reads translatable string fields from an entity.
// Iterates fieldMap directly (only translatable fields) instead of all struct fields.
func extractTranslations(entity Translatable) []Translation {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	fieldMap := entity.TranslatableFields()
	entityName := entity.TranslationEntityName()
	entityID := entity.TranslationEntityID()
	locale := entity.TranslationEntityLocale()

	results := make([]Translation, 0, len(fieldMap))
	for structFieldName, dbFieldID := range fieldMap {
		f := v.FieldByName(structFieldName)
		if !f.IsValid() || f.Kind() != reflect.String {
			continue
		}
		results = append(results, Translation{
			Entity:   entityName,
			EntityID: entityID,
			Field:    dbFieldID,
			Locale:   locale,
			Value:    f.String(),
		})
	}
	return results
}

// ------------------------------------------------
// --------------- Reflection Helpers -------------
// ------------------------------------------------

// toSnakeCase converts a string from CamelCase to snake_case.
// Rules:
// - AIRecommends  → ai_recommends
// - AICRecommends → aic_recommends
// - SomeField     → some_field
func toSnakeCase(str string) string {
	var result []rune
	runes := []rune(str)
	n := len(runes)
	for i := 0; i < n; i++ {
		if i > 0 && isUpper(runes[i]) && (i+1 < n && !isUpper(runes[i+1]) || !isUpper(runes[i-1])) {
			result = append(result, '_')
		}
		result = append(result, toLower(runes[i]))
	}
	return string(result)
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func toLower(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

// GetEntityNameFromType returns the snake_case entity name derived from a type via reflection.
// Use when you prefer not to implement TranslationEntityName() manually.
// Example: "Product" → "product", "GeoTag" → "geo_tag"
func GetEntityNameFromType[T any](t *T) string {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return toSnakeCase(typ.Name())
}

// GetEntityNameFromValue returns the snake_case entity name derived from any value via reflection.
func GetEntityNameFromValue(v any) string {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return toSnakeCase(typ.Name())
}
