package app

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/logging"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/migrator"
)

func RunMigration(cfg *config.Config) error {
	plain, err := logging.NewLogger(cfg)
	if err != nil {
		return err
	}
	return migrator.Migrate(cfg.PG.DatabaseURI, plain.Sugar())
}
