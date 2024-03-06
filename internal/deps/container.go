package deps

import (
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/logging"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/repo"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/job"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/pass"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/token"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/validator"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Container struct {
	Logger         *zap.Logger
	DB             *sqlx.DB
	AuthUseCase    *usecase.AuthUseCase
	OrderUseCase   *usecase.OrderUseCase
	BalanceUseCase *usecase.BalanceUseCase
	JobUseCase     *usecase.JobUseCase
	Config         *config.Config
	Tx             *manager.Manager
}

func NewContainer(cfg *config.Config) (*Container, error) {
	logger, err := logging.NewLogger(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Connect("pgx", cfg.PG.DatabaseURI)
	if err != nil {
		return nil, err
	}

	trm, err := manager.New(trmsqlx.NewDefaultFactory(db))
	if err != nil {
		return nil, err
	}

	getter := trmsqlx.DefaultCtxGetter
	userRepo := repo.NewUserRepo(db, getter)
	balanceRepo := repo.NewBalanceRepo(db, getter)
	orderRepo := repo.NewOrderRepo(db, getter)
	jobRepo := repo.NewJobRepo(db, getter)

	hasher := pass.NewHasher(cfg.App.PassHashCost)
	tokener := token.NewJWT([]byte(cfg.App.TokenSecretKey), cfg.App.TokenExpires)
	orderValidator := validator.NewOrderValidator()
	backgroundQueue := job.NewJobQueue(jobRepo)

	authUseCase := usecase.NewAuthUseCase(logger, userRepo, balanceRepo, hasher, tokener, trm)
	orderUseCase := usecase.NewOrderUseCase(orderRepo, balanceRepo, orderValidator, backgroundQueue, trm)
	balanceUseCase := usecase.NewBalanceUseCase(orderRepo, balanceRepo, orderValidator, trm)
	jobUseCase := usecase.NewJobUseCase(jobRepo, orderUseCase, trm)

	return &Container{
		Logger:         logger,
		DB:             db,
		AuthUseCase:    authUseCase,
		OrderUseCase:   orderUseCase,
		BalanceUseCase: balanceUseCase,
		JobUseCase:     jobUseCase,
		Tx:             trm,
		Config:         cfg,
	}, nil
}

func (c *Container) Close() {
	if err := c.DB.Close(); err != nil {
		c.Logger.Error("failed to close DB")
	}
}
