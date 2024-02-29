package pipeline

import (
	"context"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"time"
)

func (p *Pipe) poll(
	ctx context.Context,
	g *errgroup.Group,
) <-chan entity.Job {
	output := make(chan entity.Job, p.bufferSize)
	g.Go(func() error {
		defer close(output)

	loop:
		for ctx.Err() == nil {
			jobs, err := p.jobUseCase.Fetch(ctx, p.bufferSize)
			if err != nil {
				if err = p.delayOnError(ctx, err); err != nil {
					return err
				}
				continue loop
			}

			p.logger.Debug("jobs fetched", zap.Int("job_count", len(jobs)))
			for _, job := range jobs {
				select {
				case output <- job:
				case <-ctx.Done():
					p.logger.Debug("cancelled")
					break loop
				}
			}
		}

		p.logger.Debug("jobs poll shutdown", zap.Error(ctx.Err()))
		return ctx.Err()
	})

	return output
}

func (p *Pipe) delayOnError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrJobNotFound):
		p.logger.Debug("no jobs fetched")
	default:
		p.logger.Error("failed fetch jobs", zap.Error(err))
	}

	t := time.NewTimer(p.pollDelay)
	select {
	case <-p.pollTriggerCh:
		p.logger.Debug("poll triggered manually")
		return nil
	case <-t.C:
		p.logger.Debug("poll triggered by timer")
		return nil
	case <-ctx.Done():
		p.logger.Debug("cancelled")
		return ctx.Err()
	}
}
