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
		t := time.NewTimer(p.pollDelay)
		defer t.Stop()

		for ctx.Err() == nil {
			jobs, err := p.jobUseCase.Fetch(ctx, p.bufferSize)
			if err != nil {
				switch {
				case errors.Is(err, usecase.ErrJobNotFound):
					p.logger.Debug("no jobs fetched")
				default:
					p.logger.Error("failed fetch jobs", zap.Error(err))
				}

				t.Reset(p.pollDelay)
				select {
				case <-t.C:
					continue
				case <-ctx.Done():
					p.logger.Debug("cancelled")
					break
				}
			}

			p.logger.Debug("jobs fetched", zap.Int("job_count", len(jobs)))
		loop:
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
