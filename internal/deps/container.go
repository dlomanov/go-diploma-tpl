package deps

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/pkg/logging"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Container struct {
	Logger *zap.Logger
	DB     *sqlx.DB
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

	return &Container{
		Logger: logger,
		DB:     db,
	}, nil
}

func (c *Container) Close() {
	if err := c.DB.Close(); err != nil {
		c.Logger.Error("failed to close DB")
	}
}
