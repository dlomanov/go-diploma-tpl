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
	"github.com/shopspring/decimal"
)

var _ usecase.BalanceRepo = (*BalanceRepo)(nil)

type (
	BalanceRepo struct {
		db     *sqlx.DB
		getter *trmsqlx.CtxGetter
	}
	balanceRow struct {
		UserID    uuid.UUID       `db:"user_id"`
		Current   decimal.Decimal `db:"current"`
		Withdrawn decimal.Decimal `db:"withdrawn"`
		CreatedAt time.Time       `db:"created_at"`
		UpdatedAt time.Time       `db:"updated_at"`
	}
)

func NewBalanceRepo(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
) *BalanceRepo {
	return &BalanceRepo{
		db:     db,
		getter: getter,
	}
}

func (r *BalanceRepo) Get(
	ctx context.Context,
	userID entity.UserID,
) (entity.Balance, error) {
	db := r.getDB(ctx)
	bRow := balanceRow{}

	idStr := userID
	err := db.GetContext(ctx, &bRow, `
		SELECT
			user_id,
			current,
			withdrawn,
			created_at,
			updated_at
		FROM balances
		WHERE user_id = $1::uuid;`,
		idStr)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return entity.Balance{}, usecase.ErrBalanceNotFound
		default:
			return entity.Balance{}, err
		}
	}

	return bRow.toEntity(), nil
}

func (r *BalanceRepo) Save(ctx context.Context, balance entity.Balance) error {
	db := r.getDB(ctx)
	bRow := toBalanceRow(balance)

	result, err := db.NamedExecContext(ctx, `
		INSERT INTO balances (user_id, current, withdrawn, created_at, updated_at)
		VALUES (:user_id, :current, :withdrawn, :created_at, :updated_at)
		ON CONFLICT (user_id) DO NOTHING;`,
		bRow)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return usecase.ErrBalanceExists
	}

	return nil
}

func (r *BalanceRepo) Update(ctx context.Context, balance entity.Balance) error {
	db := r.getDB(ctx)
	bRow := toBalanceRow(balance)

	result, err := db.NamedExecContext(ctx, `
		UPDATE balances
		SET current = :current,
		    withdrawn = :withdrawn,
		    updated_at = :updated_at
		WHERE user_id = :user_id;`,
		bRow)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return usecase.ErrOrderNotFound
	}

	return nil
}

func (r *BalanceRepo) getDB(ctx context.Context) trmsqlx.Tr {
	return r.getter.DefaultTrOrDB(ctx, r.db)
}

func toBalanceRow(balance entity.Balance) balanceRow {
	return balanceRow{
		UserID:    uuid.UUID(balance.UserID),
		Current:   decimal.Decimal(balance.Current),
		Withdrawn: decimal.Decimal(balance.Withdrawn),
		CreatedAt: balance.CreatedAt,
		UpdatedAt: balance.UpdatedAt,
	}
}

func (row balanceRow) toEntity() entity.Balance {
	return entity.Balance{
		UserID:    entity.UserID(row.UserID),
		Current:   entity.Amount(row.Current),
		Withdrawn: entity.Amount(row.Withdrawn),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
