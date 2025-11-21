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
	Translate(ctx context.Context, locales []Lang, entities []T) ([]T, error)
}

var _ Translator[TranslatableEntity] = (*translator[TranslatableEntity])(nil)

type translator[T TranslatableEntity] struct {
	translationRepository TranslationRepository
}

/*
parms, err = r.paramTrans.Translate(ctx, locales, parms)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
*/

func NewTranslator[T TranslatableEntity](
	translationRepository TranslationRepository,
) Translator[T] {
	return &translator[T]{
		translationRepository: translationRepository,
	}
}

func (t *translator[T]) Translate(
	ctx context.Context,
	locales []Lang,
	entities []T,
) ([]T, error) {
	const op = "gotrans.Translate"

	if len(entities) == 0 {
		return nil, nil
	}

	entityType := reflect.TypeOf((*T)(nil)).Elem().Name()
	entityType = toSnakeCase(entityType)

	entityIDs := ExtractIDs(entities)
	translations, err := t.translationRepository.GetByEntityAndField(ctx, locales, entityType, entityIDs)
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
			f.SetMapIndex(reflect.ValueOf(tr.Lang), reflect.ValueOf(tr.Value))
		}
	}
	return nil
}

func ExtractIDs[T TranslatableEntity](entities []T) []int {
	ids := make([]int, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.TranslationEntityID())
	}
	return ids
}

func ExtractTranslations(entityName string, entityID int, entity any) ([]Translation, error) {
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

func ApplyTranslations(entity any, translations []Translation) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()

	fieldMap := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		fieldMap[toSnakeCase(t.Field(i).Name)] = i
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
			f.SetMapIndex(reflect.ValueOf(tr.Lang), reflect.ValueOf(tr.Value))
		}
	}
	return nil
}

type Translation struct {
	ID       int    `db:"id"`
	Entity   string `db:"entity"`
	EntityID int    `db:"entity_id"`
	Field    string `db:"field"`
	Lang     Lang   `db:"lang"`
	Value    string `db:"value"`
}

type TranslateField map[Lang]string

func (tf TranslateField) Get(lang Lang) string {
	return tf[lang]
}

func (tf TranslateField) IsEmpty() bool {
	return len(tf) == 0
}

// ------------------------------------------------
// --------------- Helpers ------------------------
// ------------------------------------------------

func filter[T any](in []T, pred func(T) bool) []T {
	var out []T
	for _, v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
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
