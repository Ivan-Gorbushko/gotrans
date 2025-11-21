package gotrans_test

import (
	"context"
	"fmt"
	"github.com/Ivan-Gorbushko/gotrans"
)

type Parameter struct {
	ID          int
	Name        gotrans.TranslateField `translatable:"true"`
	Description gotrans.TranslateField `translatable:"true"`
}

func (p Parameter) TranslationEntityID() int { return p.ID }

func ExampleTranslator() {
	var repo gotrans.TranslationRepository = &mockRepo{}

	// Creating a translator
	paramTrans := gotrans.NewTranslator[Parameter](repo)

	// Sample data
	parms := []Parameter{
		{ID: 1},
	}

	// Translation
	ctx := context.Background()
	locales := []gotrans.Lang{gotrans.LangEN, gotrans.LangRU}
	parms, err := paramTrans.Translate(ctx, locales, parms)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(parms[0].Name[gotrans.LangEN])
	// Output: Example Name EN
}

type mockRepo struct{}

func (m *mockRepo) GetByEntityAndField(ctx context.Context, locales []gotrans.Lang, entity string, entityIDs []int) ([]gotrans.Translation, error) {
	return []gotrans.Translation{
		{Entity: "parameter", EntityID: 1, Field: "name", Lang: gotrans.LangEN, Value: "Example Name EN"},
		{Entity: "parameter", EntityID: 1, Field: "name", Lang: gotrans.LangRU, Value: "Пример имени RU"},
	}, nil
}
