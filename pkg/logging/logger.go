package logging

import (
	"fmt"
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/go-errors/errors"
	"go.uber.org/zap"
)

func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(cfg.Log.Level)
	if err != nil {
		return nil, errors.New(err)
	}

	var c zap.Config
	switch cfg.Log.Type {
	case config.LoggerTypeDevelopment:
		c = zap.NewDevelopmentConfig()
	case config.LoggerTypeProduction:
		c = zap.NewDevelopmentConfig()
	default:
		return nil, fmt.Errorf("unknown logger type %s", cfg.Log.Type)
	}

	c.Level = lvl
	return c.Build()
}
