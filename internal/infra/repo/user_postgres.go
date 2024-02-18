package repo

import (
	"context"
	"database/sql"
	"errors"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ usecase.UserRepo = (*UserRepo)(nil)

type UserRepo struct {
	db     *sqlx.DB
	getter *trmsqlx.CtxGetter
}

func NewUserRepo(db *sqlx.DB, c *trmsqlx.CtxGetter) *UserRepo {
	return &UserRepo{
		db:     db,
		getter: c,
	}
}

func (r *UserRepo) Exists(ctx context.Context, login entity.Login) (result bool, err error) {
	row := r.getDB(ctx).QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1);`, login)
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

	err = r.getDB(ctx).GetContext(ctx, &model, `SELECT id, login, pass_hash FROM users WHERE login = $1;`, login)
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

func (r *UserRepo) Save(ctx context.Context, user entity.User) error {
	result, err := r.getDB(ctx).ExecContext(ctx,
		`INSERT INTO users (id, login, pass_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING;`,
		user.ID, user.Login, user.PassHash, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return usecase.ErrAuthUserExists
	}

	return nil
}

func (r *UserRepo) getDB(ctx context.Context) trmsqlx.Tr {
	return r.getter.DefaultTrOrDB(ctx, r.db)
}
