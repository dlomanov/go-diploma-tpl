package main

import (
	"log"

	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/app"
)

func main() {
	cfg := config.NewConfig()
	cfg.Print()

	if err := app.RunMigration(cfg); err != nil {
		log.Fatal(err)
	}

	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
