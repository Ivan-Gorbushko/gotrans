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
	GetByEntityAndField(
		ctx context.Context,
		locales []Locale,
		entity string,
		entityIDs []int,
	) ([]Translation, error)

	MultiCreate(
		ctx context.Context,
		translations []Translation,
	) error
}
