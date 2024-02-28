package pipeline

import "golang.org/x/sync/errgroup"

func (p *Pipe) wait(g *errgroup.Group) {
	go func() {
		defer close(p.notify)
		defer p.shutdown()
		err := g.Wait()
		p.notify <- err
	}()
}
