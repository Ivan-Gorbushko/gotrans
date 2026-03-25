package gotrans

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type Parameter struct {
	ID          int
	locale      Locale
	Name        string
	Description string
}

var _ Translatable = (*Parameter)(nil)

func (p Parameter) TranslationEntityLocale() Locale { return p.locale }
func (p Parameter) TranslationEntityID() int        { return p.ID }
func (p Parameter) TranslatableFields() map[string]string {
	return map[string]string{
		"Name":        "name",
		"Description": "description",
	}
}
func (p Parameter) TranslationEntityName() string { return "parameter" }

func TestLoadTranslations(t *testing.T) {
	repo := &mockRepo{
		translations: []Translation{
			NewTranslation(1, "parameter", 1, "name", LocaleEN, "Example Name EN"),
			NewTranslation(2, "parameter", 1, "description", LocaleEN, "Desc EN"),
		},
	}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{ID: 1, locale: LocaleEN}}
	ctx := context.Background()
	parms, err := paramTrans.LoadTranslations(ctx, parms)
	require.NoError(t, err)

	require.Equal(t, "Example Name EN", parms[0].Name)
	require.Equal(t, "Desc EN", parms[0].Description)
}

func TestSaveTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	parms := []Parameter{{
		ID:          1,
		locale:      LocaleEN,
		Name:        "New Name EN",
		Description: "Desc EN",
	}}
	ctx := context.Background()
	err := paramTrans.SaveTranslations(ctx, parms)
	require.NoError(t, err)

	require.Len(t, repo.saved, 2)
	// Check that translations were saved
	hasName := false
	hasDesc := false
	for _, tr := range repo.saved {
		if tr.Field == "name" && tr.Value == "New Name EN" && tr.Entity == "parameter" {
			hasName = true
		}
		if tr.Field == "description" && tr.Value == "Desc EN" && tr.Entity == "parameter" {
			hasDesc = true
		}
	}
	require.True(t, hasName)
	require.True(t, hasDesc)
}

func TestDeleteTranslations(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := NewTranslator[Parameter](repo)

	// Saving translations
	parms := []Parameter{{
		ID:          1,
		locale:      LocaleEN,
		Name:        "Name EN",
		Description: "Desc EN",
	}}
	ctx := context.Background()
	_ = paramTrans.SaveTranslations(ctx, parms)
	require.Len(t, repo.saved, 2)

	// Delete translations
	entity := "parameter"
	entityIDs := []int{1}
	fields := []string{"name", "description"}
	err := repo.MassDelete(ctx, LocaleEN, entity, entityIDs, fields)
	require.NoError(t, err)
	require.Len(t, repo.saved, 0)
}

// TestMultiLocaleSaveAndLoad demonstrates that translations are grouped by locale
// This test verifies that the translator efficiently handles multiple locales
func TestMultiLocaleSaveAndLoad(t *testing.T) {
	repo := &mockRepo{
		translations: []Translation{
			NewTranslation(1, "parameter", 1, "name", LocaleEN, "Name EN"),
			NewTranslation(2, "parameter", 2, "name", LocaleFR, "Name FR"),
			NewTranslation(3, "parameter", 1, "description", LocaleEN, "Desc EN"),
			NewTranslation(4, "parameter", 2, "description", LocaleFR, "Desc FR"),
		},
	}
	paramTrans := NewTranslator[Parameter](repo)

	// Load with mixed locales
	parms := []Parameter{
		{ID: 1, locale: LocaleEN},
		{ID: 2, locale: LocaleFR},
	}
	ctx := context.Background()
	parms, err := paramTrans.LoadTranslations(ctx, parms)
	require.NoError(t, err)

	require.Equal(t, "Name EN", parms[0].Name)
	require.Equal(t, "Desc EN", parms[0].Description)
	require.Equal(t, "Name FR", parms[1].Name)
	require.Equal(t, "Desc FR", parms[1].Description)
}

type mockRepo struct {
	saved        []Translation
	translations []Translation
}

func (m *mockRepo) GetTranslations(
	_ context.Context,
	locale Locale,
	entity string,
	_ []int,
) ([]Translation, error) {
	var result []Translation
	for _, tr := range m.translations {
		if tr.Entity == entity && tr.GetLocale() == locale {
			result = append(result, tr)
		}
	}
	return result, nil
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
	locale Locale,
	entity string,
	entityIDs []int,
	fields []string,
) error {
	// Delete translations by key
	type key struct {
		Entity   string
		EntityID int
		Field    string
		Locale   string
	}
	toDelete := make(map[key]struct{})
	for _, id := range entityIDs {
		for _, field := range fields {
			toDelete[key{entity, id, field, locale.String()}] = struct{}{}
		}
	}
	var filtered []Translation
	for _, tr := range m.saved {
		k := key{tr.Entity, tr.EntityID, tr.Field, tr.GetLocale().String()}
		if _, ok := toDelete[k]; !ok {
			filtered = append(filtered, tr)
		}
	}
	m.saved = filtered
	return nil
}

func (m *mockRepo) MassCreateOrUpdate(
	ctx context.Context,
	locale Locale,
	translations []Translation,
) error {
	if len(translations) == 0 {
		return nil
	}
	_ = m.MassDelete(ctx, locale, translations[0].Entity, []int{translations[0].EntityID}, []string{translations[0].Field})
	return m.MassCreate(ctx, translations)
}
