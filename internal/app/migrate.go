package app

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/pkg/logging"
	"github.com/dlomanov/go-diploma-tpl/pkg/migrator"
)

func RunMigration(cfg *config.Config) error {
	plain, err := logging.NewLogger(cfg)
	if err != nil {
		return err
	}
	return migrator.Migrate(cfg.PG.DatabaseURI, plain.Sugar())
}