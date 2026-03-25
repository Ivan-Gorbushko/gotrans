package gotrans

import (
	"context"
)

type TranslationRepository interface {
	GetTranslations(
		ctx context.Context,
		locale Locale,
		entity string,
		entityIDs []int,
	) ([]Translation, error)

	MassCreate(
		ctx context.Context,
		translations []Translation,
	) error

	MassDelete(
		ctx context.Context,
		locale Locale,
		entity string,
		entityIDs []int,
		fields []string,
	) error

	MassCreateOrUpdate(
		ctx context.Context,
		locale Locale,
		translations []Translation,
	) error
}
