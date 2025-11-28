package gotrans

import (
	"context"
	"fmt"
	"reflect"
)

type TranslatableEntity interface {
	TranslationEntityID() int
}

type Translator[T TranslatableEntity] interface {
	LoadTranslations(ctx context.Context, locales []Locale, entities []T) ([]T, error)
	SaveTranslations(ctx context.Context, entities []T) error
	DeleteTranslations(
		ctx context.Context,
		Entity string,
		EntityIDs []int,
		Fields []string,
		Locales []Locale,
	) error
}

var _ Translator[TranslatableEntity] = (*translator[TranslatableEntity])(nil)

type translator[T TranslatableEntity] struct {
	translationRepository TranslationRepository
}

func NewTranslator[T TranslatableEntity](
	translationRepository TranslationRepository,
) Translator[T] {
	return &translator[T]{
		translationRepository: translationRepository,
	}
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

	translations = filter(translations, func(tr Translation) bool {
		id := v.FieldByName("ID").Int()
		return tr.Entity == entityName && tr.EntityID == int(id)
	})

	fieldMap := make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		fieldMap[toSnakeCase(typ.Field(i).Name)] = i
	}

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

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("translatable") != "true" {
			continue
		}
		fieldName := toSnakeCase(field.Name)
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

func extractIDs[T TranslatableEntity](entities []T) []int {
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
