package gotrans

import (
	"context"
)

type Translation struct {
	ID       int    `db:"id"`
	Entity   string `db:"entity"`
	EntityID int    `db:"entity_id"`
	Field    string `db:"field"`
	Locale   Locale `db:"locale"`
	Value    string `db:"value"`
}

type TranslationRepository interface {
	GetTranslations(
		ctx context.Context,
		locales []Locale,
		entity string,
		entityIDs []int,
	) ([]Translation, error)

	MassCreate(
		ctx context.Context,
		translations []Translation,
	) error

	MassDelete(
		ctx context.Context,
		translations []Translation,
	) error

	MassCreateOrUpdate(
		ctx context.Context,
		translations []Translation,
	) error
}
