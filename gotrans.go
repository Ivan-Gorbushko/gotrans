package gotrans

import (
	"context"
	"reflect"
)

// Translatable interface for explicit association between struct fields and translation field IDs
// TranslatableFieldMap returns a map: key = struct field name, value = translation field ID in DB
// Example: map[string]string{"Title": "title", "Description": "desc", "Recommendation": "rec"}
type Translatable interface {
	TranslationEntityID() int
	TranslatableFieldMap() map[string]string
}

// Translator interface for single-locale translation operations
// All translation operations now work with a single locale and string fields
// Example usage: LoadTranslations(ctx, locale, entities)
type Translator[T Translatable] interface {
	LoadTranslations(ctx context.Context, locale Locale, entities []T) ([]T, error)
	SaveTranslations(ctx context.Context, locale Locale, entities []T) error
	DeleteTranslations(ctx context.Context, locale Locale, entity string, entityIDs []int, fields []string) error
	DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error
}

var _ Translator[Translatable] = (*translator[Translatable])(nil)

type translator[T Translatable] struct {
	translationRepository TranslationRepository
}

func (t *translator[T]) DeleteTranslationsByEntity(ctx context.Context, entity string, entityIDs []int) error {
	return t.translationRepository.MassDelete(ctx, LocaleNone, entity, entityIDs, nil)
}

func NewTranslator[T Translatable](translationRepository TranslationRepository) Translator[T] {
	return &translator[T]{
		translationRepository: translationRepository,
	}
}

func (t *translator[T]) LoadTranslations(ctx context.Context, locale Locale, entities []T) ([]T, error) {
	if len(entities) == 0 {
		return nil, nil
	}
	entityType := reflect.TypeOf((*T)(nil)).Elem().Name()
	entityType = toSnakeCase(entityType)
	entityIDs := extractIDs(entities)
	translations, err := t.translationRepository.GetTranslations(ctx, locale, entityType, entityIDs)
	if err != nil {
		return nil, err
	}
	for i := range entities {
		err = t.applyTranslations(&entities[i], locale, translations)
		if err != nil {
			return nil, err
		}
	}
	return entities, nil
}

func (t *translator[T]) SaveTranslations(ctx context.Context, locale Locale, entities []T) error {
	entityType := reflect.TypeOf((*T)(nil)).Elem().Name()
	entityName := toSnakeCase(entityType)
	var allTranslations []Translation
	for _, e := range entities {
		translations, err := extractTranslations(entityName, e.TranslationEntityID(), e, locale)
		if err != nil {
			return err
		}
		allTranslations = append(allTranslations, translations...)
	}
	if len(allTranslations) == 0 {
		return nil
	}
	return t.translationRepository.MassCreateOrUpdate(ctx, locale, allTranslations)
}

func (t *translator[T]) DeleteTranslations(ctx context.Context, locale Locale, entity string, entityIDs []int, fields []string) error {
	return t.translationRepository.MassDelete(ctx, locale, entity, entityIDs, fields)
}

// ------------------------------------------------
// --------------- Helpers ------------------------
// ------------------------------------------------

func (t *translator[T]) applyTranslations(entity *T, locale Locale, translations []Translation) error {
	v := reflect.ValueOf(entity).Elem()
	typ := v.Type()
	entityName := toSnakeCase(typ.Name())
	translatable, ok := any(entity).(Translatable)
	if !ok {
		return nil
	}
	fieldMap := translatable.TranslatableFieldMap()
	idToIndex := make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if fieldID, ok := fieldMap[name]; ok {
			idToIndex[fieldID] = i
		}
	}
	id := translatable.TranslationEntityID()
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

func extractTranslations(entityName string, entityID int, entity any, locale Locale) ([]Translation, error) {
	var results []Translation
	v := reflect.ValueOf(entity)
	if v.Kind() != reflect.Struct {
		return nil, nil
	}
	t := v.Type()
	translatable, ok := entity.(Translatable)
	if !ok {
		return nil, nil
	}
	fieldMap := translatable.TranslatableFieldMap()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
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
	return results, nil
}

func extractIDs[T Translatable](entities []T) []int {
	ids := make([]int, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.TranslationEntityID())
	}
	return ids
}

/**
 * Converts a string from CamelCase to snake_case with next rules:
 * - AIRecommends → ai_recommends
 * - AICRecommends → aic_recommends
 * - SomeField → some_field
 */
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
