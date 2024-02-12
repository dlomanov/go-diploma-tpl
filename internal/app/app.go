package app

import (
	"context"
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
)

func Run(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := deps.NewContainer(ctx, cfg)
	if err != nil {
		return err
	}
	defer c.Close()

	return nil
}
