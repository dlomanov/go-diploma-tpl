package pipeline

import (
	"context"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type (
	Pipe struct {
		logger          *zap.Logger
		bufferSize      uint
		pollDelay       time.Duration
		handlerCount    uint
		notify          chan error
		shutdown        func()
		shutdownTimeout time.Duration
		fixDelay        time.Duration
		fixProcTimeout  time.Duration
		handleMu        sync.RWMutex
		pollTriggerCh   chan struct{}
		jobUseCase      *usecase.JobUseCase
	}
)

func New(c *deps.Container) *Pipe {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pipe{
		jobUseCase:      c.JobUseCase,
		logger:          c.Logger,
		bufferSize:      c.Config.PipelineBufferSize,
		pollDelay:       c.Config.PipelinePollDelay,
		fixDelay:        c.Config.PipelineFixDelay,
		fixProcTimeout:  c.Config.PipelineFixProcTimeout,
		shutdownTimeout: c.Config.PipelineShutdownTimeout,
		notify:          make(chan error, 1),
		shutdown:        cancel,
		handleMu:        sync.RWMutex{},
		pollTriggerCh:   make(chan struct{}, 1),
	}
	g, gctx := errgroup.WithContext(ctx)

	p.fix(gctx, g)

	jobs := p.poll(gctx, g)
	p.handle(gctx, g, jobs)
	p.wait(g)

	return p
}

func (p *Pipe) Notify() <-chan error {
	return p.notify
}

func (p *Pipe) Trigger() {
	select {
	case p.pollTriggerCh <- struct{}{}:
		p.logger.Debug("poll triggered")
	default:
		p.logger.Debug("poll triggered already")
	}
}

func (p *Pipe) Shutdown() error {
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
