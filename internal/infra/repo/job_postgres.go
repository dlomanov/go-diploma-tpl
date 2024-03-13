package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	_ usecase.JobRepo = (*JobRepo)(nil)
)

type (
	JobRepo struct {
		db     *sqlx.DB
		getter *trmsqlx.CtxGetter
	}
	jobRows []jobRow
	jobRow  struct {
		ID            uuid.UUID      `db:"id"`
		Type          string         `db:"type"`
		Status        string         `db:"status"`
		EntityID      uuid.UUID      `db:"entity_id"`
		Attempt       uint           `db:"attempt"`
		LastError     sql.NullString `db:"last_error"`
		NextAttemptAt sql.NullTime   `db:"next_attempt_at"`
		CreatedAt     time.Time      `db:"created_at"`
		UpdatedAt     time.Time      `db:"updated_at"`
	}
)

func NewJobRepo(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
) *JobRepo {
	return &JobRepo{
		db:     db,
		getter: getter,
	}
}

func (r *JobRepo) Create(
	ctx context.Context,
	job entity.Job,
) error {
	db := r.getDB(ctx)

	row := toJobRow(job)
	res, err := db.NamedExecContext(ctx, `
		INSERT INTO jobs (id, type, status, entity_id, attempt, last_error, next_attempt_at, created_at, updated_at)
		VALUES (:id, :type, :status, :entity_id, :attempt, :last_error, :next_attempt_at, :created_at, :updated_at)
		ON CONFLICT (id) DO NOTHING;`, row)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return usecase.ErrJobExists
	}

	return nil
}

func (r *JobRepo) GetUpdate(
	ctx context.Context,
	count uint,
) ([]entity.Job, error) {
	db := r.getDB(ctx)

	rows := make(jobRows, 0)
	err := db.Select(&rows, `
		UPDATE jobs
		SET status = $1,
		    next_attempt_at = null,
		    updated_at = timezone('utc', now())
		WHERE id in (
			SELECT id FROM jobs
			WHERE next_attempt_at is not null
			ORDER BY next_attempt_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING
			id,
			type,
			status,
			entity_id,
			attempt,
			last_error,
			next_attempt_at,
			created_at,
			updated_at;`,
		entity.JobStatusProcessing,
		count)

	switch {
	case err != nil:
		return nil, err
	case len(rows) == 0:
		return nil, usecase.ErrJobNotFound
	default:
		return rows.toEntities(), nil
	}
}

func (r *JobRepo) Update(ctx context.Context, job entity.Job) error {
	db := r.getDB(ctx)

	result, err := db.NamedExecContext(ctx, `
		UPDATE jobs
		SET attempt = attempt + 1,
		    next_attempt_at = cast(:next_attempt_at as timestamp),
			status = cast(:status as text),
			last_error =
			    CASE
			        WHEN cast(:last_error as text) isnull THEN last_error
			        ELSE cast(:last_error as text)
			    END, 
		    updated_at = timezone('utc', now())
		WHERE id = cast(:id as uuid);`,
		toJobRow(job))

	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return usecase.ErrJobNotFound
	}

	return nil
}

func (r *JobRepo) FixProcessingTimeout(
	ctx context.Context,
	timeout time.Duration,
) error {
	db := r.getDB(ctx)

	_, err := db.ExecContext(ctx, `
		UPDATE jobs
		SET next_attempt_at = timezone('utc', now()),
			attempt = attempt + 1
		WHERE status = $1
		  AND next_attempt_at is null
		  AND timezone('utc', now()) - updated_at > $2;`,
		entity.JobStatusProcessing,
		timeout)

	return err
}

func (rows jobRows) toEntities() []entity.Job {
	res := make([]entity.Job, len(rows))
	for i, v := range rows {
		res[i] = v.toEntity()
	}
	return res
}

func (row jobRow) toEntity() entity.Job {
	model := entity.Job{
		ID:        row.ID,
		Type:      entity.JobType(row.Type),
		Status:    entity.JobStatus(row.Status),
		EntityID:  row.EntityID,
		Attempt:   row.Attempt,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.LastError.Valid {
		model.LastError = errors.New(row.LastError.String)
	}
	if row.NextAttemptAt.Valid {
		model.NextAttemptAt = row.NextAttemptAt.Time
	}

	return model
}

func toJobRow(model entity.Job) jobRow {
	row := jobRow{
		ID:        model.ID,
		Type:      string(model.Type),
		Status:    string(model.Status),
		EntityID:  model.EntityID,
		Attempt:   model.Attempt,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	if model.LastError != nil {
		row.LastError = sql.NullString{
			String: model.LastError.Error(),
			Valid:  true,
		}
	}
	if !model.NextAttemptAt.IsZero() {
		row.NextAttemptAt = sql.NullTime{
			Time:  model.NextAttemptAt,
			Valid: true,
		}
	}

	return row
}

func (r *JobRepo) getDB(ctx context.Context) trmsqlx.Tr {
	return r.getter.DefaultTrOrDB(ctx, r.db)
}
