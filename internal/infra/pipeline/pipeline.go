package pipeline

import (
	"context"
	"errors"
	"time"

	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/pipeline/stages"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	Pipeline struct {
		logger            *zap.Logger
		bufferSize        uint
		pollDelay         time.Duration
		handleWorkerCount uint
		notify            chan error
		shutdown          func()
		shutdownTimeout   time.Duration
		fixDelay          time.Duration
		fixProcTimeout    time.Duration
		stages            Stages
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
	cfg *config.Config,
) *Pipeline {
	return &Pipeline{
		logger:            logger,
		bufferSize:        cfg.PipelineBufferSize,
		pollDelay:         cfg.PipelinePollDelay,
		fixDelay:          cfg.PipelineFixDelay,
		fixProcTimeout:    cfg.PipelineFixProcTimeout,
		handleWorkerCount: cfg.PipelineHandleWorkerCount,
		shutdownTimeout:   cfg.PipelineShutdownTimeout,
		notify:            make(chan error, 1),
		shutdown:          func() { /* noop */ },
		stages: Stages{
			poll: stages.NewPollStage(
				logger,
				cfg.PipelineBufferSize,
				cfg.PipelinePollDelay),
			handle: stages.NewHandleStage(
				logger,
				cfg.PipelineHandleWorkerCount),
			fix: stages.NewFixStage(
				logger,
				cfg.PipelineFixDelay,
				cfg.PipelineFixProcTimeout),
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
