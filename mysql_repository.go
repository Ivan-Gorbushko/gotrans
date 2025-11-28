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
	Entity string,
	EntityIDs []int,
	Fields []string,
	Locales []Locale,
) error {
	const op = "translationRepository.MassDelete"

	// Basic query and arguments
	query := "DELETE FROM translations WHERE entity = ?"
	args := []any{Entity}

	if len(EntityIDs) > 0 {
		query += " AND entity_id IN (?)"
		args = append(args, EntityIDs)
	}
	if len(Fields) > 0 {
		query += " AND field IN (?)"
		args = append(args, Fields)
	}
	if len(Locales) > 0 {
		query += " AND locale IN (?)"
		args = append(args, Locales)
	}

	// TODO: Need review this logic
	// If only entity — do not delete anything
	//if len(args) == 1 {
	//	return nil
	//}

	query, args, err := sqlx.In(query, args...)
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

	// Group translations by entity
	type deleteParams struct {
		IDs     map[int]struct{}
		Fields  map[string]struct{}
		Locales map[Locale]struct{}
	}
	entityMap := make(map[string]*deleteParams)

	for _, tr := range translations {
		if _, ok := entityMap[tr.Entity]; !ok {
			entityMap[tr.Entity] = &deleteParams{
				IDs:     make(map[int]struct{}),
				Fields:  make(map[string]struct{}),
				Locales: make(map[Locale]struct{}),
			}
		}
		entityMap[tr.Entity].IDs[tr.EntityID] = struct{}{}
		entityMap[tr.Entity].Fields[tr.Field] = struct{}{}
		entityMap[tr.Entity].Locales[tr.Locale] = struct{}{}
	}

	// Remove translations for each entity
	for entity, params := range entityMap {
		var ids []int
		for id := range params.IDs {
			ids = append(ids, id)
		}
		var fields []string
		for f := range params.Fields {
			fields = append(fields, f)
		}
		var locales []Locale
		for l := range params.Locales {
			locales = append(locales, l)
		}
		if err := t.MassDelete(ctx, entity, ids, fields, locales); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	// Saving new translations
	if err := t.MassCreate(ctx, translations); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
