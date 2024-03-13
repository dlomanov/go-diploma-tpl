package pipeline

import (
	"context"
	"errors"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/infra/pipeline/stages"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	Pipeline struct {
		logger          *zap.Logger
		notify          chan error
		shutdown        func()
		shutdownTimeout time.Duration
		stages          Stages
	}
	Config struct {
		BufferSize        uint
		PollDelay         time.Duration
		FixDelay          time.Duration
		FixProcTimeout    time.Duration
		HandleWorkerCount uint
		ShutdownTimeout   time.Duration
	}
	Stages struct {
		poll   *stages.PollStage
		handle *stages.HandleStage
		fix    *stages.FixStage
	}
	Handler interface {
		stages.Fetcher
		stages.Handler
		stages.Fixer
	}
)

func New(
	logger *zap.Logger,
	cfg Config,
) *Pipeline {
	return &Pipeline{
		logger:   logger,
		notify:   make(chan error, 1),
		shutdown: func() { /* noop */ },
		stages: Stages{
			poll: stages.NewPollStage(
				logger,
				cfg.BufferSize,
				cfg.PollDelay),
			handle: stages.NewHandleStage(
				logger,
				cfg.HandleWorkerCount),
			fix: stages.NewFixStage(
				logger,
				cfg.FixDelay,
				cfg.FixProcTimeout),
		},
	}
}

func (p *Pipeline) GetPollTrigger() func() {
	return p.stages.poll.Trigger
}

func (p *Pipeline) Start(handler Handler) {
	ctx, cancel := context.WithCancel(context.Background())
	p.shutdown = cancel
	g, gctx := errgroup.WithContext(ctx)

	// independent stages
	p.stages.fix.Run(gctx, g, handler)

	// pipeline stages
	data := p.stages.poll.Run(gctx, g, handler)
	p.stages.handle.Run(gctx, g, handler, data)

	p.wait(g)
}

func (p *Pipeline) Notify() <-chan error {
	return p.notify
}

func (p *Pipeline) Shutdown() error {
	p.shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), p.shutdownTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		p.logger.Debug("pipeline shutdown timeout occurred")
		return ctx.Err()
	case err, ok := <-p.notify:
		if !ok {
			p.logger.Debug("pipeline already shutdown")
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
}

func (p *Pipeline) wait(g *errgroup.Group) {
	go func() {
		defer close(p.notify)
		defer p.shutdown()
		err := g.Wait()
		p.notify <- err
	}()
}
