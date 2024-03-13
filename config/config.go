package config

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		App        `yaml:"app"`
		Logger     `yaml:"logger"`
		PG         `yaml:"postgres"`
		Server     `yaml:"server"`
		AccrualAPI `yaml:"accrual_api"`
		Pipeline   `yaml:"pipeline"`
	}

	App struct {
		PassHashCost   int           `yaml:"pass_hash_cost,omitempty" env:"PASS_HASH_COST"`
		TokenSecretKey string        `yaml:"token_secret_key,omitempty" env:"TOKEN_SECRET_KEY"`
		TokenExpires   time.Duration `yaml:"token_expires" env:"TOKEN_EXPIRES"`
	}

	Logger struct {
		LoggerType string `yaml:"type" env:"LOGGER_TYPE"`
		LogLevel   string `yaml:"level" env:"LOG_LEVEL"`
	}

	PG struct {
		DatabaseURI string `yaml:"url" env:"DATABASE_URI"`
	}

	Server struct {
		ServerAddr            string        `yaml:"address" env:"RUN_ADDRESS"`
		ServerShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT"`
	}

	AccrualAPI struct {
		AccrualAPIAddr string `yaml:"address" env:"ACCRUAL_SYSTEM_ADDRESS"`
	}

	Pipeline struct {
		PipelineBufferSize        uint          `yaml:"buffer_size" env:"PIPELINE_BUFFER_SIZE"`
		PipelineHandleWorkerCount uint          `yaml:"handler_count" env:"PIPELINE_HANDLER_COUNT"`
		PipelineShutdownTimeout   time.Duration `yaml:"shutdown_timeout" env:"PIPELINE_SHUTDOWN_TIMEOUT"`
		PipelinePollDelay         time.Duration `yaml:"poll_delay" env:"PIPELINE_POLL_DELAY"`
		PipelineFixDelay          time.Duration `yaml:"fix_delay" env:"PIPELINE_FIX_DELAY"`
		PipelineFixProcTimeout    time.Duration `yaml:"fix_proc_timeout" env:"PIPELINE_FIX_PROC_TIMEOUT"`
	}
)

//go:embed config.yml
var configFile embed.FS

func New() *Config {
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
		log.Fatal(err)
	}

	f := flag.NewFlagSet("", flag.ContinueOnError)
	f.StringVar(&cfg.Server.ServerAddr, "a", cfg.Server.ServerAddr, "service run address and port")
	f.StringVar(&cfg.AccrualAPI.AccrualAPIAddr, "r", cfg.AccrualAPI.AccrualAPIAddr, "accrual system address")
	f.StringVar(&cfg.PG.DatabaseURI, "d", cfg.PG.DatabaseURI, "postgres database uri")
	err := f.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if err = cleanenv.ReadEnv(cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}

func NewFromYAML() *Config {
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
		log.Fatal(err)
	}
	return cfg
}

func (c Config) Print() {
	c.TokenSecretKey = ""
	c.PassHashCost = 0

	cStr, err := yaml.Marshal(c)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("config:\n%s\n", cStr)
}
