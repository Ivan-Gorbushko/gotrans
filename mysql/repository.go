package mysql

import (
	"context"
	"fmt"

	"github.com/ivan-gorbushko/gotrans"
	"github.com/jmoiron/sqlx"
)

type translationRepository struct {
	db *sqlx.DB
}

var _ gotrans.TranslationRepository = (*translationRepository)(nil)

func NewTranslationRepository(db *sqlx.DB) gotrans.TranslationRepository {
	return &translationRepository{db: db}
}

func (t *translationRepository) GetTranslations(
	ctx context.Context,
	locale gotrans.Locale,
	entity string,
	entityIDs []int,
) ([]gotrans.Translation, error) {
	const op = "translationRepository.GetByEntityAndField"
	const batchSize = 1000

	var allMysqlTranslations []Translation

	for start := 0; start < len(entityIDs); start += batchSize {
		end := start + batchSize
		if end > len(entityIDs) {
			end = len(entityIDs)
		}
		batchIDs := entityIDs[start:end]

		query, args, err := sqlx.In(`
			SELECT * FROM translations WHERE entity = ? AND locale = ? AND entity_id IN (?)
		`, entity, locale.String(), batchIDs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		query = t.db.Rebind(query)

		var translations []Translation
		err = t.db.SelectContext(ctx, &translations, query, args...)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		allMysqlTranslations = append(allMysqlTranslations, translations...)
	}

	// Converting to domain repository model
	allTranslations := make([]gotrans.Translation, len(allMysqlTranslations))
	for i := range allMysqlTranslations {
		allTranslations[i] = toTranslateModel(allMysqlTranslations[i])
	}

	return allTranslations, nil
}

func (t *translationRepository) MassCreate(
	ctx context.Context,
	translations []gotrans.Translation,
) error {
	const op = "translationRepository.MassCreate"
	if len(translations) == 0 {
		return nil
	}

	mysqlTranslations := make([]Translation, len(translations))
	for i := range translations {
		mysqlTranslations[i] = toMysqlTranslateModel(translations[i])
	}

	insertQuery := `
		INSERT INTO translations (entity, entity_id, field, locale, value)
		VALUES (:entity, :entity_id, :field, :locale, :value)
	`
	_, err := t.db.NamedExecContext(ctx, insertQuery, mysqlTranslations)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (t *translationRepository) MassDelete(
	ctx context.Context,
	locale gotrans.Locale,
	entity string,
	entityIDs []int,
	fields []string,
) error {
	const op = "translationRepository.MassDelete"

	query := "DELETE FROM translations WHERE entity = ?"
	args := []any{entity}

	if locale != gotrans.LocaleNone {
		query += " AND locale = ?"
		args = append(args, locale.String())
	}
	if len(entityIDs) > 0 {
		query += " AND entity_id IN (?)"
		args = append(args, entityIDs)
	}
	if len(fields) > 0 {
		query += " AND field IN (?)"
		args = append(args, fields)
	}

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

func (t *translationRepository) MassCreateOrUpdate(
	ctx context.Context,
	locale gotrans.Locale,
	translations []gotrans.Translation,
) error {
	const op = "translationRepository.MassCreateOrUpdate"
	if len(translations) == 0 {
		return nil
	}

	// Group translations by entity
	type deleteParams struct {
		IDs    map[int]struct{}
		Fields map[string]struct{}
	}
	entityMap := make(map[string]*deleteParams)

	for _, tr := range translations {
		if _, ok := entityMap[tr.Entity]; !ok {
			entityMap[tr.Entity] = &deleteParams{
				IDs:    make(map[int]struct{}),
				Fields: make(map[string]struct{}),
			}
		}
		entityMap[tr.Entity].IDs[tr.EntityID] = struct{}{}
		entityMap[tr.Entity].Fields[tr.Field] = struct{}{}
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
		if err := t.MassDelete(ctx, locale, entity, ids, fields); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	mysqlTranslations := make([]Translation, len(translations))
	for i := range translations {
		mysqlTranslations[i] = toMysqlTranslateModel(translations[i])
	}

	if err := t.MassCreate(ctx, translations); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func toMysqlTranslateModel(tr gotrans.Translation) Translation {
	return Translation{
		ID:       tr.ID,
		Entity:   tr.Entity,
		EntityID: tr.EntityID,
		Field:    tr.Field,
		Locale:   tr.Locale.String(),
		Value:    tr.Value,
	}
}

func toTranslateModel(mt Translation) gotrans.Translation {
	locale, ok := gotrans.ParseLocale(mt.Locale)
	if !ok {
		locale = gotrans.LocaleNone
	}
	return gotrans.Translation{
		ID:       mt.ID,
		Entity:   mt.Entity,
		EntityID: mt.EntityID,
		Field:    mt.Field,
		Locale:   locale,
		Value:    mt.Value,
	}
}
