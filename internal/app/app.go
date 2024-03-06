package app

import (
	"github.com/dlomanov/go-diploma-tpl/config"
	v1 "github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/v1"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/httpserver"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/pipeline"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func Run(cfg *config.Config) error {
	c, err := deps.NewContainer(cfg)
	if err != nil {
		return err
	}
	defer c.Close()

	c.Logger.Info("run app")

	s := startServer(c)
	p := startPipeline(c)
	wait(c, s, p)
	shutdownServer(c, s)
	shutdownPipeline(c, p)

	c.Logger.Debug("app terminated")
	return nil
}

func startServer(c *deps.Container) *httpserver.Server {
	r := chi.NewRouter()
	v1.NewRouter(r, c)
	s := httpserver.New(
		r,
		httpserver.Addr(c.Config.ServerAddr),
		httpserver.ShutdownTimeout(c.Config.ServerShutdownTimeout))
	c.Logger.Debug("server started")

	return s
}

func shutdownServer(c *deps.Container, s *httpserver.Server) {
	c.Logger.Debug("server shutdown")
	if err := s.Shutdown(); err != nil {
		c.Logger.Error("server shutdown error", zap.Error(err))
		return
	}
	c.Logger.Debug("server shutdown - ok")
}

func startPipeline(c *deps.Container) *pipeline.Pipeline {
	p := c.StartPipeline()
	c.Logger.Debug("pipeline started")
	return p
}

func shutdownPipeline(c *deps.Container, p *pipeline.Pipeline) {
	c.Logger.Debug("pipeline shutdown")
	if err := p.Shutdown(); err != nil {
		c.Logger.Error("pipeline shutdown - error", zap.Error(err))
		return
	}
	c.Logger.Debug("pipeline shutdown - ok")
}

func wait(
	c *deps.Container,
	server *httpserver.Server,
	pipe *pipeline.Pipeline,
) {
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-terminate:
		c.Logger.Info("cached terminate signal", zap.String("signal", s.String()))
	case err := <-pipe.Notify():
		c.Logger.Error("pipe notified error", zap.Error(err))
	case err := <-server.Notify():
		c.Logger.Error("server notified error", zap.Error(err))
	}
}
