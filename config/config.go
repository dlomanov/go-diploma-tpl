package config

import (
	"embed"
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"io/fs"
	"os"
)

type (
	Config struct {
		Log         `yaml:"logger"`
		PG          `yaml:"postgres"`
		HTTP        `yaml:"http"`
		AccrualHTTP `yaml:"accrual_http"`
	}

	Log struct {
		Type  LoggerType `yaml:"type" env:"LOG_TYPE"`
		Level string     `yaml:"level" env:"LOG_LEVEL"`
	}
	LoggerType string

	PG struct {
		DatabaseURI string `yaml:"url" env:"DATABASE_URI"`
	}

	HTTP struct {
		RunAddress string `yaml:"address" env:"RUN_ADDRESS"`
	}

	AccrualHTTP struct {
		Address string `yaml:"address" env:"ACCRUAL_SYSTEM_ADDRESS"`
	}
)

const (
	LoggerTypeDevelopment LoggerType = "development"
	LoggerTypeProduction  LoggerType = "production"
)

//go:embed config.yml
var configFile embed.FS

func NewConfig() (*Config, error) {
	cfg := new(Config)

	parseFile := func() error {
		file, err := configFile.Open("config.yml")
		if err != nil {
			return err
		}
		defer func(f fs.File) { _ = f.Close() }(file)

		return cleanenv.ParseYAML(file, cfg)
	}
	if err := parseFile(); err != nil {
		return nil, err
	}

	f := flag.NewFlagSet("", flag.ContinueOnError)
	f.StringVar(&cfg.HTTP.RunAddress, "a", cfg.HTTP.RunAddress, "service run address and port")
	f.StringVar(&cfg.AccrualHTTP.Address, "r", cfg.AccrualHTTP.Address, "accrual system address")
	f.StringVar(&cfg.PG.DatabaseURI, "d", cfg.PG.DatabaseURI, "postgres database uri")
	err := f.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	if err = cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
