package config

import (
	"embed"
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"io/fs"
	"os"
	"time"
)

type (
	Config struct {
		App         `yaml:"app"`
		Logger      `yaml:"logger"`
		PG          `yaml:"postgres"`
		HTTP        `yaml:"http"`
		AccrualHTTP `yaml:"accrual_http"`
		Pipeline    `yaml:"pipeline"`
	}

	App struct {
		PassHashCost   int           `yaml:"pass_hash_cost" env:"PASS_HASH_COST"`
		TokenSecretKey string        `yaml:"token_secret_key" env:"TOKEN_SECRET_KEY"`
		TokenExpires   time.Duration `yaml:"token_expires" env:"TOKEN_EXPIRES"`
	}

	Logger struct {
		LoggerType LoggerType `yaml:"type" env:"LOGGER_TYPE"`
		LogLevel   string     `yaml:"level" env:"LOG_LEVEL"`
	}
	LoggerType string

	PG struct {
		DatabaseURI string `yaml:"url" env:"DATABASE_URI"`
	}

	HTTP struct {
		RunAddress string `yaml:"address" env:"RUN_ADDRESS"`
	}

	AccrualHTTP struct {
		AccrualAddress string `yaml:"address" env:"ACCRUAL_SYSTEM_ADDRESS"`
	}

	Pipeline struct {
		PipelineBufferSize      uint          `yaml:"buffer_size" env:"PIPELINE_BUFFER_SIZE"`
		PipelineShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"PIPELINE_SHUTDOWN_TIMEOUT"`
		PipelinePollDelay       time.Duration `yaml:"poll_delay" env:"PIPELINE_POLL_DELAY"`
		PipelineFixDelay        time.Duration `yaml:"fix_delay" env:"PIPELINE_FIX_DELAY"`
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
	f.StringVar(&cfg.AccrualHTTP.AccrualAddress, "r", cfg.AccrualHTTP.AccrualAddress, "accrual system address")
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
