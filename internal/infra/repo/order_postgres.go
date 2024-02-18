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
	"github.com/shopspring/decimal"
	"time"
)

var _ usecase.OrderRepo = (*OrderRepo)(nil)

type (
	OrderRepo struct {
		db     *sqlx.DB
		getter *trmsqlx.CtxGetter
	}
	orderRows []orderRow
	orderRow  struct {
		ID        uuid.UUID       `db:"id"`
		Number    string          `db:"number"`
		Type      string          `db:"type"`
		Status    string          `db:"status"`
		Income    decimal.Decimal `db:"income"`
		Outcome   decimal.Decimal `db:"outcome"`
		UserID    uuid.UUID       `db:"user_id"`
		CreatedAt time.Time       `db:"created_at"`
		UpdatedAt time.Time       `db:"updated_at"`
	}
)

func NewOrderRepo(db *sqlx.DB, c *trmsqlx.CtxGetter) *OrderRepo {
	return &OrderRepo{
		db:     db,
		getter: c,
	}
}

func (r *OrderRepo) Get(
	ctx context.Context,
	id entity.OrderID,
) (entity.Order, error) {

	db := r.getDB(ctx)
	oRow := orderRow{}

	err := db.GetContext(ctx, &oRow, `
		SELECT
			id,
			number,
			type,
			status,
			income,
			outcome,
			user_id,
			created_at,
			updated_at
		FROM orders
		WHERE id = $1;`,
		id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return entity.Order{}, usecase.ErrOrderNotFound
		default:
			return entity.Order{}, err
		}
	}

	return oRow.toEntity()
}

func (r *OrderRepo) GetAll(ctx context.Context, filter *usecase.OrderFilter) ([]entity.Order, error) {
	db := r.getDB(ctx)
	oRows := make(orderRows, 0)

	const argsCount = 2
	args := make([]any, 0, argsCount)
	switch {
	case filter != nil:
		if filter.Type != nil {
			args = append(args, *filter.Type)
		}
		if filter.UserID != nil {
			args = append(args, *filter.UserID)
		}
	default:
		args = append(args,
			sql.NullString{},
			uuid.NullUUID{})
	}

	err := db.SelectContext(ctx, &oRows, `
		SELECT
			id,
			number,
			type,
			status,
			income,
			outcome,
			user_id,
			created_at,
			updated_at
		FROM orders
		WHERE ($1 isnull OR $1 = type)
		  AND ($2 isnull OR $2 = user_id);`, args)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return make([]entity.Order, 0), nil
		default:
			return nil, err
		}
	}

	return oRows.toEntities()
}

func (r *OrderRepo) Save(ctx context.Context, order entity.Order) error {
	db := r.getDB(ctx)
	row, err := toOrderRow(order)
	if err != nil {
		return err
	}

	res, err := db.NamedExecContext(ctx, `
		INSERT INTO orders (id, number, type, status, income, outcome, user_id, created_at, updated_at)
		VALUES (:id, :number, :type, :status, :income, :outcome, :user_id, :created_at, :updated_at)
		ON CONFLICT (number) DO NOTHING;`,
		&row)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return usecase.ErrOrderExists
	}

	return nil
}

func (r *OrderRepo) Update(ctx context.Context, order entity.Order) error {
	db := r.getDB(ctx)
	row, err := toOrderRow(order)
	if err != nil {
		return err
	}

	res, err := db.NamedExecContext(ctx, `
		UPDATE orders
		SET status = :status,
		    income = :income,
		    outcome = :outcome,
		    updated_at = :updated_at
		WHERE id = :id;`,
		&row)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return usecase.ErrOrderNotFound
	}

	return nil
}

func (r *OrderRepo) getDB(ctx context.Context) trmsqlx.Tr {
	return r.getter.DefaultTrOrDB(ctx, r.db)
}

func toOrderRow(order entity.Order) (orderRow, error) {
	row := orderRow{
		ID:        uuid.UUID(order.ID),
		Number:    string(order.Number),
		Type:      string(order.Type),
		Status:    string(order.Status),
		UserID:    uuid.UUID(order.UserID),
		CreatedAt: order.CreatedAt,
		UpdatedAt: order.UpdatedAt,
	}
	switch order.Type {
	case entity.OrderTypeIncome:
		row.Income = decimal.Decimal(order.Amount)
	case entity.OrderTypeOutcome:
		row.Outcome = decimal.Decimal(order.Amount)
	default:
		return orderRow{}, usecase.ErrOrderTypeInvalid
	}

	return row, nil
}

func (rows orderRows) toEntities() ([]entity.Order, error) {
	orders := make([]entity.Order, 0, len(rows))
	for _, oRow := range rows {
		order, err := oRow.toEntity()
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (row orderRow) toEntity() (entity.Order, error) {
	order := entity.Order{
		ID:        entity.OrderID(row.ID),
		Number:    entity.OrderNumber(row.Number),
		Type:      entity.OrderType(row.Type),
		Status:    entity.OrderStatus(row.Status),
		UserID:    entity.UserID(row.UserID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Events:    entity.NewEvents(),
	}
	if !order.Status.Valid() {
		return entity.Order{}, usecase.ErrOrderStatusInvalid
	}
	switch order.Type {
	case entity.OrderTypeIncome:
		order.Amount = entity.Amount(row.Income)
	case entity.OrderTypeOutcome:
		order.Amount = entity.Amount(row.Outcome)
	default:
		return entity.Order{}, usecase.ErrOrderTypeInvalid
	}

	return order, nil
}