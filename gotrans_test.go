package gotrans_test

import (
	"context"
	"github.com/Ivan-Gorbushko/gotrans"
	"testing"

	"github.com/stretchr/testify/require"
)

type Parameter struct {
	ID          int
	Name        gotrans.TranslateField `translatable:"true"`
	Description gotrans.TranslateField `translatable:"true"`
}

func (p Parameter) TranslationEntityID() int { return p.ID }

func TestReadFromTranslator(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := gotrans.NewTranslator[Parameter](repo)

	parms := []Parameter{{ID: 1}}
	ctx := context.Background()
	locales := []gotrans.Locale{gotrans.LocaleEN, gotrans.LocaleRU}
	parms, err := paramTrans.Translate(ctx, locales, parms)
	require.NoError(t, err)

	require.Equal(t, "Example Name EN", parms[0].Name[gotrans.LocaleEN])
	require.Equal(t, "Пример имени RU", parms[0].Name[gotrans.LocaleRU])
}

func TestWriteToTranslator(t *testing.T) {
	repo := &mockRepo{}
	paramTrans := gotrans.NewTranslator[Parameter](repo)

	parms := []Parameter{{ID: 1}}
	ctx := context.Background()
	locales := []gotrans.Locale{gotrans.LocaleEN, gotrans.LocaleRU}
	parms, err := paramTrans.Translate(ctx, locales, parms)
	require.NoError(t, err)

	require.Equal(t, "Example Name EN", parms[0].Name[gotrans.LocaleEN])
	require.Equal(t, "Пример имени RU", parms[0].Name[gotrans.LocaleRU])
}

type mockRepo struct{}

func (m *mockRepo) GetByEntityAndField(
	_ context.Context,
	_ []gotrans.Locale,
	entity string,
	entityIDs []int,
) ([]gotrans.Translation, error) {
	return []gotrans.Translation{
		{Entity: entity, EntityID: 1, Field: "name", Locale: gotrans.LocaleEN, Value: "Example Name EN"},
		{Entity: entity, EntityID: 1, Field: "name", Locale: gotrans.LocaleRU, Value: "Пример имени RU"},
	}, nil
}
