package pipeline

import (
	"context"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"time"
)

func (p *Pipe) handle(
	ctx context.Context,
	g *errgroup.Group,
	input <-chan entity.Job,
) {
	var (
		outer           = g
		inner, innerCtx = errgroup.WithContext(ctx)
	)

	outer.Go(func() error {

		for id := uint(1); id <= p.handlerCount; id++ {
			workerID := id
			inner.Go(func() error {
				return p.worker(innerCtx, workerID, input)
			})
		}

		return inner.Wait()
	})
}

func (p *Pipe) worker(
	ctx context.Context,
	id uint,
	input <-chan entity.Job,
) error {
	var (
		workerID = zap.Uint("worker_id", id)
		job      entity.Job
		open     bool
	)

	for ctx.Err() == nil {
		select {
		case job, open = <-input:
			if !open {
				p.logger.Debug("input chan closed", workerID)
				break
			}
		case <-ctx.Done():
			p.logger.Debug("cancelled", workerID)
			break
		}

		if err := p.retry(ctx, job); err != nil {
			return err
		}

	}
	p.logger.Debug("handle shutdown", zap.Error(ctx.Err()), workerID)
	return ctx.Err()
}

func (p *Pipe) retry(
	ctx context.Context,
	job entity.Job,
) error {
	var (
		rerr *entity.RetryError
		t    *time.Timer
	)
	defer func(t *time.Timer) {
		if t != nil {
			t.Stop()
		}
	}(t)

	for {
		err := p.jobUseCase.Handle(ctx, job)
		switch {
		case errors.As(err, &rerr):
			if t != nil {
				t.Reset(rerr.RetryAfter)
			} else {
				t = time.NewTimer(rerr.RetryAfter)
			}
			select {
			case <-t.C:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		case err != nil:
			if err = p.jobUseCase.Fail(ctx, job, err); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}
