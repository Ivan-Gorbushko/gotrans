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

func (t *mysqlTranslationRepository) GetTranslations(
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

func (t *mysqlTranslationRepository) MassCreate(
	ctx context.Context,
	translations []Translation,
) error {
	const op = "translationRepository.MassCreate"
	if len(translations) == 0 {
		return nil
	}

	// Inserting new translations
	insertQuery := `
		INSERT INTO translations (entity, entity_id, field, locale, value)
		VALUES (:entity, :entity_id, :field, :locale, :value)
	`
	_, err := t.db.NamedExecContext(ctx, insertQuery, translations)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (t *mysqlTranslationRepository) MassDelete(
	ctx context.Context,
	translations []Translation,
) error {
	const op = "translationRepository.MassDelete"
	if len(translations) == 0 {
		return nil
	}

	// Collect unique keys for deletion
	type key struct {
		Entity   string
		EntityID int
		Field    string
		Locale   Locale
	}
	keys := make(map[key]struct{})
	for _, tr := range translations {
		keys[key{tr.Entity, tr.EntityID, tr.Field, tr.Locale}] = struct{}{}
	}

	// Forming slices for mass deletion
	var entities []string
	var entityIDs []int
	var fields []string
	var locales []Locale
	for k := range keys {
		entities = append(entities, k.Entity)
		entityIDs = append(entityIDs, k.EntityID)
		fields = append(fields, k.Field)
		locales = append(locales, k.Locale)
	}

	// Mass deletion
	query, args, err := sqlx.In(`
		DELETE FROM translations
		WHERE entity IN (?) AND entity_id IN (?) AND field IN (?) AND locale IN (?)
	`, entities, entityIDs, fields, locales)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	query = t.db.Rebind(query)

	_, err = t.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (t *mysqlTranslationRepository) MassCreateOrUpdate(
	ctx context.Context,
	translations []Translation,
) error {
	const op = "translationRepository.MassCreateOrUpdate"
	if len(translations) == 0 {
		return nil
	}

	err := t.MassDelete(ctx, translations)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = t.MassCreate(ctx, translations)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
