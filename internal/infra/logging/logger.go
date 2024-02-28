package logging

import (
	"fmt"
	"github.com/dlomanov/go-diploma-tpl/config"
	"go.uber.org/zap"
)

func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(cfg.Logger.LogLevel)
	if err != nil {
		return nil, err
	}

	var c zap.Config
	switch cfg.Logger.LoggerType {
	case config.LoggerTypeDevelopment:
		c = zap.NewDevelopmentConfig()
	case config.LoggerTypeProduction:
		c = zap.NewProductionConfig()
	default:
		return nil, fmt.Errorf("unknown logger type %s", cfg.Logger.LoggerType)
	}

	c.Level = lvl
	return c.Build()
}
