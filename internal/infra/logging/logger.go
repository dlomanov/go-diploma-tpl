package logging

import (
	"fmt"

	"go.uber.org/zap"
)

const (
	LoggerTypeDevelopment LoggerType = "development"
	LoggerTypeProduction  LoggerType = "production"
)

type (
	Config struct {
		Level string
		Type  string
	}
	LoggerType string
)

func NewLogger(cfg Config) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	var c zap.Config
	switch LoggerType(cfg.Type) {
	case LoggerTypeDevelopment:
		c = zap.NewDevelopmentConfig()
	case LoggerTypeProduction:
		c = zap.NewProductionConfig()
	default:
		return nil, fmt.Errorf("unknown logger type %s", cfg.Type)
	}

	c.Level = lvl
	return c.Build()
}
