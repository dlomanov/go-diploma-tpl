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
	"time"
)

var _ usecase.UserRepo = (*UserRepo)(nil)

type (
	UserRepo struct {
		db     *sqlx.DB
		getter *trmsqlx.CtxGetter
	}
	userRow struct {
		ID        uuid.UUID `db:"id"`
		Login     string    `db:"login"`
		PassHash  string    `db:"pass_hash"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
)

func NewUserRepo(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
) *UserRepo {
	return &UserRepo{
		db:     db,
		getter: getter,
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
	db := r.getDB(ctx)
	row := userRow{}

	err = db.GetContext(ctx, &row, `SELECT id, login, pass_hash FROM users WHERE login = $1;`, login)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return user, usecase.ErrAuthUserNotFound
		default:
			return user, err
		}
	}

	return r.toEntity(row), nil
}

func (r *UserRepo) Create(ctx context.Context, user entity.User) error {
	db := r.getDB(ctx)
	row := r.toRow(user)

	result, err := db.NamedExecContext(ctx, `
		INSERT INTO users (id, login, pass_hash, created_at, updated_at)
		VALUES (:id, :login, :pass_hash, :created_at, :updated_at)
		ON CONFLICT (id) DO NOTHING;`,
		row)
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

func (*UserRepo) toRow(user entity.User) userRow {
	return userRow{
		ID:        uuid.UUID(user.ID),
		Login:     string(user.Login),
		PassHash:  string(user.PassHash),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (*UserRepo) toEntity(row userRow) entity.User {
	return entity.User{
		ID: entity.UserID(row.ID),
		HashCreds: entity.HashCreds{
			Login:    entity.Login(row.Login),
			PassHash: entity.PassHash(row.PassHash),
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
