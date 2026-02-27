package pool

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/time/rate"
)

type Job struct {
	ID   int
	Task func(ctx context.Context) error
}

type Result struct {
	JobID int
	Err   error
}

type Pool struct {
	jobs        chan Job
	results     chan Result
	limiter     *rate.Limiter
	wg          sync.WaitGroup
	workerCount int
	submitted   atomic.Int64
	completed   atomic.Int64
	cancel      context.CancelFunc
}

func New(workerCount int, limiter rate.Limit, burst int) *Pool {
	return &Pool{
		jobs:        make(chan Job, workerCount),
		results:     make(chan Result, workerCount),
		limiter:     rate.NewLimiter(limiter, burst),
		workerCount: workerCount,
	}
}

func (p *Pool) Start(ctx context.Context) context.Context {
	ctx, p.cancel = context.WithCancel(ctx)

	for range p.workerCount {
		p.wg.Add(1)
		go p.worker(ctx)
	}

	return ctx
}

func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}

			if err := p.limiter.Wait(ctx); err != nil {
				p.results <- Result{JobID: job.ID, Err: err}
				p.completed.Add(1)
				continue
			}

			err := job.Task(ctx)
			p.results <- Result{JobID: job.ID, Err: err}
			p.completed.Add(1)
		}
	}
}

func (p *Pool) Submit(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.jobs <- job:
		p.submitted.Add(1)
		return nil
	}
}

func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

func (p *Pool) Cancel() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *Pool) Results() <-chan Result {
	return p.results
}

func (p *Pool) Submitted() int64 {
	return p.submitted.Load()
}

func (p *Pool) Completed() int64 {
	return p.completed.Load()
}
