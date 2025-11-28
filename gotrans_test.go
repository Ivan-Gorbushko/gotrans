package gotrans

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type Parameter struct {
	ID          int
	Name        TranslateField `translatable:"true"`
	Description TranslateField `translatable:"true"`
}

func (p Parameter) TranslationEntityID() int { return p.ID }

func TestLoadTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{ID: 1}}
	ctx := context.Background()
	locales := []Locale{LocaleEN, LocaleRU}
	parms, err := paramTrans.LoadTranslations(ctx, locales, parms)
	require.NoError(t, err)

	require.Equal(t, "Example Name EN", parms[0].Name[LocaleEN])
	require.Equal(t, "Пример имени RU", parms[0].Name[LocaleRU])
}

func TestSaveTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{
		ID: 1,
		Name: TranslateField{
			LocaleEN: "New Name EN",
			LocaleRU: "Новое имя RU",
		},
		Description: TranslateField{
			LocaleEN: "Desc EN",
			LocaleRU: "Описание RU",
		},
	}}
	ctx := context.Background()
	err := paramTrans.SaveTranslations(ctx, parms)
	require.NoError(t, err)

	require.Len(t, repo.saved, 4)
	require.Contains(t, repo.saved, Translation{
		Entity:   "parameter",
		EntityID: 1,
		Field:    "name",
		Locale:   LocaleEN,
		Value:    "New Name EN",
	})
	require.Contains(t, repo.saved, Translation{
		Entity:   "parameter",
		EntityID: 1,
		Field:    "name",
		Locale:   LocaleRU,
		Value:    "Новое имя RU",
	})
	require.Contains(t, repo.saved, Translation{
		Entity:   "parameter",
		EntityID: 1,
		Field:    "description",
		Locale:   LocaleEN,
		Value:    "Desc EN",
	})
	require.Contains(t, repo.saved, Translation{
		Entity:   "parameter",
		EntityID: 1,
		Field:    "description",
		Locale:   LocaleRU,
		Value:    "Описание RU",
	})
}

func TestDeleteTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	// Сохраняем переводы
	parms := []Parameter{{
		ID: 1,
		Name: TranslateField{
			LocaleEN: "Name EN",
			LocaleRU: "Имя RU",
		},
		Description: TranslateField{
			LocaleEN: "Desc EN",
			LocaleRU: "Описание RU",
		},
	}}
	ctx := context.Background()
	err := paramTrans.SaveTranslations(ctx, parms)
	require.NoError(t, err)
	require.Len(t, repo.saved, 4)

	// Удаляем переводы
	entity := "parameter"
	entityIDs := []int{1}
	fields := []string{"name", "description"}
	locales := []Locale{LocaleEN, LocaleRU}
	err = repo.MassDelete(ctx, entity, entityIDs, fields, locales)
	require.NoError(t, err)
	require.Len(t, repo.saved, 0)
}

type mockRepo struct {
	saved []Translation
}

func (m *mockRepo) GetTranslations(
	_ context.Context,
	_ []Locale,
	entity string,
	entityIDs []int,
) ([]Translation, error) {
	return []Translation{
		{Entity: entity, EntityID: 1, Field: "name", Locale: LocaleEN, Value: "Example Name EN"},
		{Entity: entity, EntityID: 1, Field: "name", Locale: LocaleRU, Value: "Пример имени RU"},
	}, nil
}

func (m *mockRepo) MassCreate(
	_ context.Context,
	translations []Translation,
) error {
	m.saved = append(m.saved, translations...)
	return nil
}

func (m *mockRepo) MassDelete(
	_ context.Context,
	entity string,
	entityIDs []int,
	fields []string,
	locales []Locale,
) error {
	// Удаляем переводы по ключу
	type key struct {
		Entity   string
		EntityID int
		Field    string
		Locale   Locale
	}
	toDelete := make(map[key]struct{})
	for _, id := range entityIDs {
		for _, field := range fields {
			for _, locale := range locales {
				toDelete[key{entity, id, field, locale}] = struct{}{}
			}
		}
	}
	var filtered []Translation
	for _, tr := range m.saved {
		k := key{tr.Entity, tr.EntityID, tr.Field, tr.Locale}
		if _, ok := toDelete[k]; !ok {
			filtered = append(filtered, tr)
		}
	}
	m.saved = filtered
	return nil
}

func (m *mockRepo) MassCreateOrUpdate(
	ctx context.Context,
	translations []Translation,
) error {
	// Группируем по entity, entityID, field, locale
	entityMap := make(map[string]map[int]map[string]map[Locale]string)
	for _, tr := range translations {
		if _, ok := entityMap[tr.Entity]; !ok {
			entityMap[tr.Entity] = make(map[int]map[string]map[Locale]string)
		}
		if _, ok := entityMap[tr.Entity][tr.EntityID]; !ok {
			entityMap[tr.Entity][tr.EntityID] = make(map[string]map[Locale]string)
		}
		if _, ok := entityMap[tr.Entity][tr.EntityID][tr.Field]; !ok {
			entityMap[tr.Entity][tr.EntityID][tr.Field] = make(map[Locale]string)
		}
		entityMap[tr.Entity][tr.EntityID][tr.Field][tr.Locale] = tr.Value
	}
	// Собираем параметры для MassDelete
	for entity, ids := range entityMap {
		var entityIDs []int
		var fields []string
		var locales []Locale
		idSet := make(map[int]struct{})
		fieldSet := make(map[string]struct{})
		localeSet := make(map[Locale]struct{})
		for id, flds := range ids {
			idSet[id] = struct{}{}
			for field, locs := range flds {
				fieldSet[field] = struct{}{}
				for locale := range locs {
					localeSet[locale] = struct{}{}
				}
			}
		}
		for id := range idSet {
			entityIDs = append(entityIDs, id)
		}
		for field := range fieldSet {
			fields = append(fields, field)
		}
		for locale := range localeSet {
			locales = append(locales, locale)
		}
		_ = m.MassDelete(ctx, entity, entityIDs, fields, locales)
	}
	return m.MassCreate(ctx, translations)
}
