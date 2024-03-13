package main

import (
	"context"
	"log"

	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/app"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/logging"
)

func main() {
	ctx := context.Background()
	cfg := config.New()
	cfg.Print()

	logger, err := logging.NewLogger(logging.Config{
		Level: cfg.Logger.LogLevel,
		Type:  cfg.LoggerType,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := app.RunMigration(logger, cfg); err != nil {
		log.Fatal(err)
	}

	if err := app.Run(ctx, logger, cfg); err != nil {
		log.Fatal(err)
	}
}
