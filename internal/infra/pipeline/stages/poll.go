package stages

import (
	"context"
	"errors"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	PollStage struct {
		logger     *zap.Logger
		bufferSize uint
		delay      time.Duration
		triggerCh  chan struct{}
	}
	Fetcher interface {
		Fetch(ctx context.Context, count uint) ([]entity.Job, error)
	}
)

func NewPollStage(
	logger *zap.Logger,
	bufferSize uint,
	delay time.Duration,
) *PollStage {
	return &PollStage{
		logger:     logger,
		bufferSize: bufferSize,
		delay:      delay,
		triggerCh:  make(chan struct{}, 1),
	}
}

func (s *PollStage) Run(
	ctx context.Context,
	g *errgroup.Group,
	fetcher Fetcher,
) <-chan entity.Job {
	output := make(chan entity.Job, s.bufferSize)
	g.Go(func() error {
		defer close(output)

		s.logger.Debug("started")
	loop:
		for ctx.Err() == nil {
			jobs, err := fetcher.Fetch(ctx, s.bufferSize)
			if err != nil {
				s.report(err)
				if err = s.wait(ctx); err != nil {
					return err
				}
				continue loop
			}

			s.logger.Debug("jobs fetched", zap.Int("job_count", len(jobs)))
			for _, job := range jobs {
				select {
				case output <- job:
				case <-ctx.Done():
					s.logger.Debug("catch cancellation")
					break loop
				}
			}
		}

		s.logger.Debug("shutdown", zap.Error(ctx.Err()))
		return ctx.Err()
	})

	return output
}

func (s *PollStage) Trigger() {
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

func (s *PollStage) report(err error) {
	var errNoData *apperrors.AppErrorNotFound
	switch {
	case errors.As(err, &errNoData):
		s.logger.Debug("no data")
	case err != nil:
		s.logger.Error("fetch jobs failed", zap.Error(err))
	}
}

func (s *PollStage) wait(ctx context.Context) error {
	t := time.NewTimer(s.delay)
	select {
	case <-s.triggerCh:
		s.logger.Debug("triggered manually")
		return nil
	case <-t.C:
		// s.logger.Debug("triggered by timer")
		return nil
	case <-ctx.Done():
		s.logger.Debug("catch cancellation")
		return ctx.Err()
	}
}
