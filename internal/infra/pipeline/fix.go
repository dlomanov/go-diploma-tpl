package pipeline

import (
	"context"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"time"
)

func (p *Pipe) fix(ctx context.Context, g *errgroup.Group) {
	g.Go(func() error {
		timer := time.NewTimer(p.fixProcDelay)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
		}
		for ctx.Err() == nil {
			err := p.jobUseCase.FixProcessing(ctx)
			if err != nil {
				p.logger.Error("fix processing failed", zap.Error(err))
			}

			select {
			case <-timer.C:
			case <-ctx.Done():
			}
		}
		return ctx.Err()
	})
}
