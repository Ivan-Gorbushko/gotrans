package mysql

import (
	"context"
	"fmt"

	"github.com/Ivan-Gorbushko/gotrans"
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
	locales []gotrans.Locale,
	entity string,
	entityIDs []int,
) ([]gotrans.Translation, error) {
	const op = "translationRepository.GetByEntityAndField"
	const batchSize = 1000

	var allMysqlTranslations []Translation
	mysqlLocales := make([]string, len(locales))
	for i, l := range locales {
		mysqlLocales[i] = l.String()
	}

	for start := 0; start < len(entityIDs); start += batchSize {
		end := start + batchSize
		if end > len(entityIDs) {
			end = len(entityIDs)
		}
		batchIDs := entityIDs[start:end]

		query, args, err := sqlx.In(`
			SELECT * FROM translations WHERE entity = ? AND locale IN (?) AND entity_id IN (?)
		`, entity, mysqlLocales, batchIDs)
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

	// Inserting new translations
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
	Entity string,
	EntityIDs []int,
	Fields []string,
	Locales []gotrans.Locale,
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

func (t *translationRepository) MassCreateOrUpdate(
	ctx context.Context,
	translations []gotrans.Translation,
) error {
	const op = "translationRepository.MassCreateOrUpdate"
	if len(translations) == 0 {
		return nil
	}

	// Group translations by entity
	type deleteParams struct {
		IDs     map[int]struct{}
		Fields  map[string]struct{}
		Locales map[gotrans.Locale]struct{}
	}
	entityMap := make(map[string]*deleteParams)

	for _, tr := range translations {
		if _, ok := entityMap[tr.Entity]; !ok {
			entityMap[tr.Entity] = &deleteParams{
				IDs:     make(map[int]struct{}),
				Fields:  make(map[string]struct{}),
				Locales: make(map[gotrans.Locale]struct{}),
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
		var locales []gotrans.Locale
		for l := range params.Locales {
			locales = append(locales, l)
		}
		if err := t.MassDelete(ctx, entity, ids, fields, locales); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	// Converting to mysql models
	mysqlTranslations := make([]Translation, len(translations))
	for i := range translations {
		mysqlTranslations[i] = toMysqlTranslateModel(translations[i])
	}

	// Saving new translations
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
