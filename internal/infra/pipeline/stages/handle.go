package stages

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	HandleStage struct {
		logger      *zap.Logger
		workerCount uint
		mu          sync.RWMutex
	}
	Handler interface {
		Handle(ctx context.Context, job entity.Job) error
		Fail(ctx context.Context, job entity.Job, err error) error
	}
)

func NewHandleStage(
	logger *zap.Logger,
	workerCount uint,
) *HandleStage {
	return &HandleStage{
		logger:      logger,
		workerCount: workerCount,
	}
}

func (s *HandleStage) Run(
	ctx context.Context,
	g *errgroup.Group,
	handler Handler,
	input <-chan entity.Job,
) {
	var (
		inner, innerCtx = errgroup.WithContext(ctx)
	)

	g.Go(func() error {
		s.logger.Debug("started")

		for id := uint(1); id <= s.workerCount; id++ {
			workerID := id
			inner.Go(func() error {
				return s.worker(innerCtx, workerID, handler, input)
			})
		}

		err := inner.Wait()
		s.logger.Debug("shutdown", zap.Error(err))
		return err
	})
}

func (s *HandleStage) worker(
	ctx context.Context,
	id uint,
	handler Handler,
	input <-chan entity.Job,
) error {
	var (
		workerID = zap.Uint("worker_id", id)
		job      entity.Job
		open     bool
		handle   = func(job entity.Job) error {
			jobID := zap.String("job_id", job.ID.String())
			err := s.retry(ctx, func() error {
				return handler.Handle(ctx, job)
			})
			if err != nil {
				s.logger.Debug("handler returns err", workerID, jobID, zap.Error(err))
				return handler.Fail(ctx, job, err)
			}
			return nil
		}
	)

	s.logger.Debug("worker started", workerID)
loop:
	for ctx.Err() == nil {
		select {
		case job, open = <-input:
			if !open {
				s.logger.Debug("input chan closed", workerID)
				break loop
			}
		case <-ctx.Done():
			s.logger.Debug("catch cancellation", workerID)
			break loop
		}
		jobID := zap.String("job_id", job.ID.String())
		s.logger.Debug("got job", workerID, jobID)

		if err := handle(job); err != nil {
			s.logger.Error("failed to handle job", workerID, jobID, zap.Error(err))
			return err
		}
		s.logger.Debug("job handled", workerID, jobID)
	}

	s.logger.Debug("worker shutdown", zap.Error(ctx.Err()), workerID)
	return ctx.Err()
}

func (s *HandleStage) retry(
	ctx context.Context,
	f func() error,
) error {
	var (
		rerr   *apperrors.AppErrorTransient
		handle = func() error {
			err := s.withRLock(f)
			return err
		}
		wait = func() error {
			return s.withTryLock(func() error {
				t := time.NewTimer(rerr.RetryAfter)
				defer t.Stop()

				select {
				case <-t.C:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
		}
	)

	for {
		err := handle()
		switch {
		case errors.As(err, &rerr):
			s.logger.Debug("delay workers", zap.Duration("delay", rerr.RetryAfter))
			if err = wait(); err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			return nil
		}
	}
}

func (s *HandleStage) withRLock(f func() error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return f()
}

func (s *HandleStage) withTryLock(f func() error) error {
	if !s.mu.TryLock() {
		s.logger.Debug("mutex already acquired")
		return nil
	}
	defer s.mu.Unlock()
	return f()
}
