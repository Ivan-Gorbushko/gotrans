package gotrans

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type mysqlTranslationRepository struct {
	db *sqlx.DB
}

var _ TranslationRepository = (*mysqlTranslationRepository)(nil)

func NewTranslationRepository(db *sqlx.DB) TranslationRepository {
	return &mysqlTranslationRepository{db: db}
}

func (t *mysqlTranslationRepository) GetByEntityAndField(
	ctx context.Context,
	locales []Locale,
	entity string,
	entityIDs []int,
) ([]Translation, error) {
	const op = "translationRepository.GetByEntityAndField"
	const batchSize = 1000

	var allTranslations []Translation

	for start := 0; start < len(entityIDs); start += batchSize {
		end := start + batchSize
		if end > len(entityIDs) {
			end = len(entityIDs)
		}
		batchIDs := entityIDs[start:end]

		query, args, err := sqlx.In(`
			SELECT * FROM translations WHERE entity = ? AND locale IN (?) AND entity_id IN (?)
		`, entity, locales, batchIDs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		query = t.db.Rebind(query)

		var translations []Translation
		err = t.db.SelectContext(ctx, &translations, query, args...)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		allTranslations = append(allTranslations, translations...)
	}

	return allTranslations, nil
}
