package gotrans

import (
	"context"
)

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
		Entity string,
		EntityIDs []int,
		Fields []string,
		Locales []Locale,
	) error

	MassCreateOrUpdate(
		ctx context.Context,
		translations []Translation,
	) error
}
