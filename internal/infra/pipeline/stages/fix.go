package stages

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type (
	FixStage struct {
		logger            *zap.Logger
		delay             time.Duration
		processingTimeout time.Duration
	}
	Fixer interface {
		FixProcessingTimeout(ctx context.Context, timeout time.Duration) error
	}
)

func NewFixStage(
	logger *zap.Logger,
	delay time.Duration,
	processingTimeout time.Duration,
) *FixStage {
	return &FixStage{
		logger:            logger,
		delay:             delay,
		processingTimeout: processingTimeout,
	}
}

func (s *FixStage) Run(ctx context.Context, g *errgroup.Group, fixer Fixer) {
	g.Go(func() error {
		s.logger.Debug("started")
		timer := time.NewTimer(s.delay)
		defer timer.Stop()

		for ctx.Err() == nil {
			err := fixer.FixProcessingTimeout(ctx, s.processingTimeout)
			if err != nil {
				s.logger.Error("iter - failed", zap.Error(err))
			} else {
				s.logger.Debug("iter - ok")
			}

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(s.delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				s.logger.Debug("catch cancellation")
			}
		}
		s.logger.Debug("shutdown", zap.Error(ctx.Err()))
		return ctx.Err()
	})
}
