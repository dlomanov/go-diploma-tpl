package repo

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
)

var _ usecase.UserRepo = (*UserRepo)(nil)

type UserRepo struct {
	db *sqlx.DB
}

func NewUser(db *sqlx.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (r *UserRepo) Exists(ctx context.Context, login entity.Login) (result bool, err error) {
	row := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1);`, login)
	if err = row.Err(); err != nil {
		return false, err
	}

	if err = row.Scan(&result); err != nil {
		return false, err
	}

	return result, nil
}

func (r *UserRepo) Get(ctx context.Context, login entity.Login) (user entity.User, err error) {
	model := struct {
		ID       uuid.UUID `db:"id"`
		Login    string    `db:"login"`
		PassHash string    `db:"pass_hash"`
	}{}

	err = r.db.GetContext(ctx, &model, `SELECT id, login, pass_hash FROM users WHERE login = $1;`, login)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return user, usecase.ErrAuthUserNotFound
		default:
			return user, err
		}
	}

	user.ID = entity.UserID(model.ID)
	user.Login = entity.Login(model.Login)
	user.PassHash = entity.PassHash(model.PassHash)
	return user, nil
}

func (r *UserRepo) Create(ctx context.Context, creds entity.HashCreds) (entity.UserID, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return entity.UserID{}, err
	}

	created := time.Now().UTC()

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, login, pass_hash, created_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING;`,
		id, creds.Login, creds.PassHash, created)
	if err != nil {
		return entity.UserID{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return entity.UserID{}, err
	}
	if affected == 0 {
		return entity.UserID{}, usecase.ErrAuthUserExists
	}

	return entity.UserID(id), nil
}
