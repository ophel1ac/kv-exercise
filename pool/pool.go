package pool

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type Job struct {
	Task func(ctx context.Context) error
}

type Pool struct {
	jobs        chan Job
	limiter     *rate.Limiter
	wg          *sync.WaitGroup
	workerCount int
}

func New(workerCount int) *Pool {
	return &Pool{
		jobs:        make(chan Job, workerCount),
		limiter:     rate.NewLimiter(rate.Limit(10), 10),
		wg:          &sync.WaitGroup{},
		workerCount: workerCount,
	}
}

func Start(ctx context.Context) {}

func Submit(ctx context.Context, job Job) {}

func Stop(ctx context.Context) {}

func Wait(ctx context.Context) {}
