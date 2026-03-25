package gotrans

import (
	"context"
	"reflect"
)

// Translatable interface for explicit association between struct fields and translation field IDs
// TranslatableFields returns a map: key = struct field name, value = translation field ID in DB
// Example: map[string]string{"Title": "title", "Description": "desc", "Recommendation": "rec"}
// TranslationEntityName returns the name of the entity as stored in translations table
type Translatable interface {
	TranslationEntityID() int
	TranslationEntityName() string
	TranslationEntityLocale() Locale
	TranslatableFields() map[string]string
}

// Translator interface for single-locale translation operations
// All translation operations now work with locale from entity (via TranslationEntityLocale() method)
// Example usage: LoadTranslations(ctx, entities)
type Translator[T Translatable] interface {
	LoadTranslations(ctx context.Context, entities []T) ([]T, error)
	SaveTranslations(ctx context.Context, entities []T) error
	DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error
	DeleteTranslations(
		ctx context.Context,
		locale Locale,
		entity string,
		entityIDs []int,
		fields []string,
	) error
}

var _ Translator[Translatable] = (*translator[Translatable])(nil)

type translator[T Translatable] struct {
	translationRepository TranslationRepository
}

func NewTranslator[T Translatable](translationRepository TranslationRepository) Translator[T] {
	return &translator[T]{
		translationRepository: translationRepository,
	}
}

func (t *translator[T]) DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error {
	return t.translationRepository.MassDelete(ctx, LocaleNone, entity, entityIDs, nil)
}

func (t *translator[T]) LoadTranslations(ctx context.Context, entities []T) ([]T, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	// Get entity name from first entity
	entityName := entities[0].TranslationEntityName()

	// Group entities by locale for optimized loading
	localeMap := make(map[Locale][]int)
	for _, e := range entities {
		locale := e.TranslationEntityLocale()
		localeMap[locale] = append(localeMap[locale], e.TranslationEntityID())
	}

	// Load translations for each locale group
	var allTranslations []Translation
	for locale, entityIDs := range localeMap {
		translations, err := t.translationRepository.GetTranslations(ctx, locale, entityName, entityIDs)
		if err != nil {
			return nil, err
		}
		allTranslations = append(allTranslations, translations...)
	}

	// Apply translations to entities
	for i := range entities {
		err := t.applyTranslations(&entities[i], allTranslations)
		if err != nil {
			return nil, err
		}
	}

	return entities, nil
}

func (t *translator[T]) SaveTranslations(ctx context.Context, entities []T) error {
	if len(entities) == 0 {
		return nil
	}

	// Group translations by locale for batch save
	localeMap := make(map[Locale][]Translation)
	for _, e := range entities {
		translations := extractTranslations(e)
		locale := e.TranslationEntityLocale()
		localeMap[locale] = append(localeMap[locale], translations...)
	}

	// Save grouped translations for each locale
	for locale, translations := range localeMap {
		if len(translations) == 0 {
			continue
		}
		if err := t.translationRepository.MassCreateOrUpdate(ctx, locale, translations); err != nil {
			return err
		}
	}

	return nil
}

func (t *translator[T]) DeleteTranslations(ctx context.Context, locale Locale, entity string, entityIDs []int, fields []string) error {
	return t.translationRepository.MassDelete(ctx, locale, entity, entityIDs, fields)
}

// ------------------------------------------------
// --------------- Helpers ------------------------
// ------------------------------------------------

func (t *translator[T]) applyTranslations(entity *T, translations []Translation) error {
	v := reflect.ValueOf(entity).Elem()
	typ := v.Type()
	translatable, ok := any(entity).(Translatable)
	if !ok {
		return nil
	}
	entityName := translatable.TranslationEntityName()
	fieldMap := translatable.TranslatableFields()
	
	// Build field index once
	idToIndex := make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if fieldID, ok := fieldMap[name]; ok {
			idToIndex[fieldID] = i
		}
	}
	
	id := translatable.TranslationEntityID()
	locale := translatable.TranslationEntityLocale()
	
	for _, tr := range translations {
		if tr.Entity != entityName || tr.EntityID != id || tr.Locale != locale {
			continue
		}
		idx, ok := idToIndex[tr.Field]
		if !ok {
			continue
		}
		f := v.Field(idx)
		if f.Kind() == reflect.String && f.CanSet() {
			f.SetString(tr.Value)
		}
	}
	return nil
}

func extractTranslations(entity Translatable) []Translation {
	var results []Translation
	
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return results
	}
	
	typ := v.Type()
	entityName := entity.TranslationEntityName()
	entityID := entity.TranslationEntityID()
	locale := entity.TranslationEntityLocale()
	fieldMap := entity.TranslatableFields()
	
	for i := 0; i < v.NumField(); i++ {
		field := typ.Field(i)
		name := field.Name
		fieldID, ok := fieldMap[name]
		if !ok {
			continue
		}
		f := v.Field(i)
		if f.Kind() == reflect.String {
			results = append(results, Translation{
				Entity:   entityName,
				EntityID: entityID,
				Field:    fieldID,
				Locale:   locale,
				Value:    f.String(),
			})
		}
	}
	return results
}

// toSnakeCase converts a string from CamelCase to snake_case.
// Rules:
// - AIRecommends → ai_recommends
// - AICRecommends → aic_recommends
// - SomeField → some_field
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

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func toLower(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

// ------------------------------------------------
// --------------- Reflection Helpers -----------
// ------------------------------------------------

// GetEntityNameFromType returns the snake_case entity name from a type.
// This is a helper function for when you don't want to implement TranslationEntityName()
// Example: "Product" → "product", "GeoTag" → "geo_tag"
func GetEntityNameFromType[T any](t *T) string {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return toSnakeCase(typ.Name())
}

// GetEntityNameFromValue returns the snake_case entity name from a value.
// This is a helper function for when you don't want to implement TranslationEntityName()
func GetEntityNameFromValue(v any) string {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return toSnakeCase(typ.Name())
}
