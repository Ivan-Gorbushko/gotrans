package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	const op = "translationRepository.GetTranslations"
	const batchSize = 1000

	var all []Translation
	for start := 0; start < len(entityIDs); start += batchSize {
		end := start + batchSize
		if end > len(entityIDs) {
			end = len(entityIDs)
		}
		query, args, err := sqlx.In(
			`SELECT * FROM translations WHERE entity = ? AND locale = ? AND entity_id IN (?)`,
			entity, locale.String(), entityIDs[start:end],
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		var batch []Translation
		if err = t.db.SelectContext(ctx, &batch, t.db.Rebind(query), args...); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		all = append(all, batch...)
	}

	result := make([]gotrans.Translation, len(all))
	for i, mt := range all {
		result[i] = toTranslateModel(mt)
	}
	return result, nil
}

func (t *translationRepository) MassDelete(
	ctx context.Context,
	locale gotrans.Locale,
	entity string,
	entityIDs []int,
	fields []string,
) error {
	const op = "translationRepository.MassDelete"
	if err := t.massDelete(ctx, t.db, locale, entity, entityIDs, fields); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// MassCreateOrUpdate deletes existing translations for the affected
// (entity, entityID, field) combinations and inserts the new ones,
// all within a single transaction to guarantee atomicity.
func (t *translationRepository) MassCreateOrUpdate(
	ctx context.Context,
	locale gotrans.Locale,
	translations []gotrans.Translation,
) error {
	const op = "translationRepository.MassCreateOrUpdate"
	if len(translations) == 0 {
		return nil
	}

	// Collect the set of (entity → IDs, fields) to delete before inserting.
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

	tx, err := t.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback() //nolint:errcheck

	for entity, params := range entityMap {
		ids := make([]int, 0, len(params.IDs))
		for id := range params.IDs {
			ids = append(ids, id)
		}
		fields := make([]string, 0, len(params.Fields))
		for f := range params.Fields {
			fields = append(fields, f)
		}
		if err = t.massDelete(ctx, tx, locale, entity, ids, fields); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	rows := make([]Translation, len(translations))
	for i, tr := range translations {
		rows[i] = toMysqlTranslateModel(tr)
	}
	if err = massInsert(ctx, tx, rows); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return tx.Commit()
}

// ------------------------------------------------
// --------------- Private helpers ----------------
// ------------------------------------------------

// dbExec is satisfied by both *sqlx.DB and *sqlx.Tx.
type dbExec interface {
	Rebind(string) string
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// massDelete is the internal delete helper used by both MassDelete and MassCreateOrUpdate.
// exec accepts either *sqlx.DB or *sqlx.Tx.
func (t *translationRepository) massDelete(
	ctx context.Context,
	exec dbExec,
	locale gotrans.Locale,
	entity string,
	entityIDs []int,
	fields []string,
) error {
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
		return err
	}
	_, err = exec.ExecContext(ctx, exec.Rebind(query), args...)
	return err
}

// massInsert performs a single bulk INSERT for all rows using the provided transaction.
// Rows are split into batches of insertBatchSize to stay within driver parameter limits.
func massInsert(ctx context.Context, tx *sqlx.Tx, rows []Translation) error {
	const insertBatchSize = 500 // 500 rows × 5 cols = 2500 params, safe for MySQL and SQLite
	for start := 0; start < len(rows); start += insertBatchSize {
		end := start + insertBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[start:end]

		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*5)
		for i, r := range batch {
			placeholders[i] = "(?, ?, ?, ?, ?)"
			args = append(args, r.Entity, r.EntityID, r.Field, r.Locale, r.Value)
		}

		query := "INSERT INTO translations (entity, entity_id, field, locale, value) VALUES " +
			strings.Join(placeholders, ", ")
		if _, err := tx.ExecContext(ctx, tx.Rebind(query), args...); err != nil {
			return err
		}
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
