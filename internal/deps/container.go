package deps

import (
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/algo/pass"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/algo/token"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/logging"
	repo2 "github.com/dlomanov/go-diploma-tpl/internal/infra/repo"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Container struct {
	Logger       *zap.Logger
	DB           *sqlx.DB
	AuthUseCase  *usecase.AuthUseCase
	OrderUseCase *usecase.OrderUseCase
	JobUseCase   *usecase.JobUseCase
	Config       *config.Config
	Tx           *manager.Manager
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

	userRepo := repo2.NewUserRepo(db, trmsqlx.DefaultCtxGetter)
	balanceRepo := repo2.NewBalanceRepo(db, trmsqlx.DefaultCtxGetter)
	hasher := pass.NewHasher(cfg.App.PassHashCost)
	tokener := token.NewJWT([]byte(cfg.App.TokenSecretKey), cfg.App.TokenExpires)
	authUseCase := usecase.NewAuthUseCase(userRepo, balanceRepo, hasher, tokener, trm)

	return &Container{
		Logger:      logger,
		DB:          db,
		AuthUseCase: authUseCase,
		Tx:          trm,
		Config:      cfg,
	}, nil
}

func (c *Container) Close() {
	if err := c.DB.Close(); err != nil {
		c.Logger.Error("failed to close DB")
	}
}
