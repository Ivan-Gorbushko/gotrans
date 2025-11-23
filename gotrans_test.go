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

type mockRepo struct {
	saved []Translation
}

func (m *mockRepo) GetByEntityAndField(
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

func (m *mockRepo) MultiCreate(
	_ context.Context,
	translations []Translation,
) error {
	m.saved = append(m.saved, translations...)
	return nil
}
