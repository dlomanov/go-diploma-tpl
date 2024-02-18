package app

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/v1"
	httpserver2 "github.com/dlomanov/go-diploma-tpl/internal/infra/httpserver"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(cfg *config.Config) error {
	c, err := deps.NewContainer(cfg)
	if err != nil {
		return err
	}
	defer c.Close()

	log := c.Logger

	r := chi.NewRouter()
	v1.NewRouter(r, c)
	server := httpserver2.New(
		r,
		httpserver2.Addr(cfg.HTTP.RunAddress),
		httpserver2.ShutdownTimeout(15*time.Second))
	log.Debug("server started")

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-terminate:
		log.Error("cached terminate signal", zap.String("signal", s.String()))
	case err = <-server.Notify():
		log.Error("server notified error", zap.Error(err))
	}

	log.Debug("server shutdown")
	if err = server.Shutdown(); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	log.Debug("app terminated")
	return nil
}
