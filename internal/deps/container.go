package deps

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/pkg/logging"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/pass"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/repo"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/token"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Container struct {
	Logger      *zap.Logger
	DB          *sqlx.DB
	Tokener     usecase.Tokener
	AuthUseCase *usecase.AuthUseCase
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

	userRepo := repo.NewUser(db)
	hasher := pass.NewHasher(cfg.App.PassHashCost)
	tokener := token.NewJWT([]byte(cfg.App.TokenSecretKey), cfg.App.TokenExpires)
	authUseCase := usecase.NewAuth(userRepo, hasher, tokener)

	return &Container{
		Logger:      logger,
		DB:          db,
		Tokener:     tokener,
		AuthUseCase: authUseCase,
	}, nil
}

func (c *Container) Close() {
	if err := c.DB.Close(); err != nil {
		c.Logger.Error("failed to close DB")
	}
}
