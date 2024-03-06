package pipeline

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func (p *Pipe) fix(ctx context.Context, g *errgroup.Group) {
	g.Go(func() error {
		timer := time.NewTimer(p.fixDelay)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
		}
		for ctx.Err() == nil {
			err := p.jobUseCase.FixProcessing(ctx, p.fixProcTimeout)
			if err != nil {
				p.logger.Error("fix processing failed", zap.Error(err))
			}

			timer.Reset(p.fixDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
			}
		}
		return ctx.Err()
	})
}
