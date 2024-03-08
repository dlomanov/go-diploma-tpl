package app

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/migrator"
	"go.uber.org/zap"
)

func RunMigration(logger *zap.Logger, cfg *config.Config) error {
	logger.Info("run migrations")
	return migrator.Migrate(cfg.PG.DatabaseURI, logger.Sugar())
}
