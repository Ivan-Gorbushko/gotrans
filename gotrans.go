package gotrans

import (
	"context"
	"fmt"
	"reflect"
)

// Translatable Interface for explicit specification of translatable fields
// TranslatableFields returns a list of field names to be translated
// Example: []string{"Title", "Description", "Recommendation"}
type Translatable interface {
	TranslationEntityID() int
	TranslatableFields() []string
}

type Translator[T Translatable] interface {
	LoadTranslations(ctx context.Context, locales []Locale, entities []T) ([]T, error)
	SaveTranslations(ctx context.Context, entities []T) error
	DeleteTranslations(
		ctx context.Context,
		Entity string,
		EntityIDs []int,
		Fields []string,
		Locales []Locale,
	) error
	SupportedLocales() []Locale
}

var _ Translator[Translatable] = (*translator[Translatable])(nil)

type translator[T Translatable] struct {
	locales               []Locale
	translationRepository TranslationRepository
}

func NewTranslator[T Translatable](
	locales []Locale,
	translationRepository TranslationRepository,
) Translator[T] {
	return &translator[T]{
		locales:               locales,
		translationRepository: translationRepository,
	}
}

func (t *translator[T]) SupportedLocales() []Locale {
	return t.locales
}

func (t *translator[T]) LoadTranslations(
	ctx context.Context,
	locales []Locale,
	entities []T,
) ([]T, error) {
	const op = "translator.LoadTranslations"

	if len(entities) == 0 {
		return nil, nil
	}

	entityType := reflect.TypeOf((*T)(nil)).Elem().Name()
	entityType = toSnakeCase(entityType)

	entityIDs := extractIDs(entities)
	translations, err := t.translationRepository.GetTranslations(
		ctx,
		locales,
		entityType,
		entityIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range entities {
		err = t.applyTranslations(&entities[i], translations)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	return entities, nil
}

func (t *translator[T]) SaveTranslations(
	ctx context.Context,
	entities []T,
) error {
	const op = "translator.SaveTranslations"

	entityType := reflect.TypeOf((*T)(nil)).Elem().Name()
	entityName := toSnakeCase(entityType)

	var allTranslations []Translation
	for _, e := range entities {
		translations, err := extractTranslations(entityName, e.TranslationEntityID(), e)
		if err != nil {
			return err
		}
		allTranslations = append(allTranslations, translations...)
	}

	if len(allTranslations) == 0 {
		return nil
	}

	err := t.translationRepository.MassCreateOrUpdate(ctx, allTranslations)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (t *translator[T]) DeleteTranslations(
	ctx context.Context,
	Entity string,
	EntityIDs []int,
	Fields []string,
	Locales []Locale,
) error {
	const op = "translator.DeleteTranslations"

	err := t.translationRepository.MassDelete(
		ctx,
		Entity,
		EntityIDs,
		Fields,
		Locales,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

type TranslateField map[Locale]string

func (tf TranslateField) Get(locale Locale) string {
	return tf[locale]
}

func (tf TranslateField) IsEmpty() bool {
	return len(tf) == 0
}

// ------------------------------------------------
// --------------- Helpers ------------------------
// ------------------------------------------------

func (t *translator[T]) applyTranslations(entity *T, translations []Translation) error {
	v := reflect.ValueOf(entity).Elem()
	typ := v.Type()
	entityName := toSnakeCase(typ.Name())

	// Get translatable fields via Translatable interface
	translatable, ok := any(entity).(Translatable)
	if !ok {
		return nil // No translatable fields
	}
	fields := translatable.TranslatableFields()

	fieldMap := make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		for _, f := range fields {
			if name == f {
				fieldMap[toSnakeCase(name)] = i
			}
		}
	}

	// Filter translations for current entity only
	id := translatable.TranslationEntityID()
	translations = filter(translations, func(tr Translation) bool {
		return tr.Entity == entityName && tr.EntityID == id
	})

	for _, tr := range translations {
		idx, ok := fieldMap[tr.Field]
		if !ok {
			continue
		}
		f := v.Field(idx)
		if f.Kind() == reflect.Map && f.Type().AssignableTo(reflect.TypeOf(TranslateField{})) {
			if f.IsNil() {
				f.Set(reflect.MakeMap(f.Type()))
			}
			f.SetMapIndex(reflect.ValueOf(tr.Locale), reflect.ValueOf(tr.Value))
		}
	}
	return nil
}

func extractTranslations(entityName string, entityID int, entity any) ([]Translation, error) {
	var results []Translation

	v := reflect.ValueOf(entity)
	if v.Kind() != reflect.Struct {
		return nil, nil
	}
	t := v.Type()

	// Get translatable fields via Translatable interface
	translatable, ok := entity.(Translatable)
	if !ok {
		return nil, nil
	}
	fields := translatable.TranslatableFields()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		name := field.Name
		isTranslatable := false
		for _, f := range fields {
			if name == f {
				isTranslatable = true
				break
			}
		}
		if !isTranslatable {
			continue
		}
		fieldName := toSnakeCase(name)
		fieldValue := v.Field(i).Interface()
		tf, ok := fieldValue.(TranslateField)
		if !ok {
			continue
		}
		for locale, value := range tf {
			results = append(results, Translation{
				Entity:   entityName,
				EntityID: entityID,
				Field:    fieldName,
				Locale:   locale,
				Value:    value,
			})
		}
	}
	return results, nil
}

func filter[T any](in []T, pred func(T) bool) []T {
	var out []T
	for _, v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
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
