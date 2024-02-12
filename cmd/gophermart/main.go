package main

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/app"
	"log"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v\n", cfg)

	log.Println("run migrations")
	if err = app.RunMigration(cfg); err != nil {
		log.Fatal(err)
	}

	log.Println("run app")
	if err = app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
