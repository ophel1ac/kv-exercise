package pool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/time/rate"
)

func TestPool_BasicExecution(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(4, rate.Inf, 1)
		ctx := p.Start(t.Context())

		const jobCount = 20
		results := make(map[int]bool)

		go func() {
			for r := range p.Results() {
				if r.Err != nil {
					t.Errorf("job %d failed: %v", r.JobID, r.Err)
				}
				results[r.JobID] = true
			}
		}()

		for i := 0; i < jobCount; i++ {
			id := i
			_ = p.Submit(ctx, Job{
				ID: id,
				Task: func(ctx context.Context) error {
					return nil
				},
			})
		}

		p.Stop()
		synctest.Wait()

		if len(results) != jobCount {
			t.Fatalf("expected %d results, got %d", jobCount, len(results))
		}
	})
}

func TestPool_ConcurrentSubmit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(8, rate.Inf, 1)
		ctx := p.Start(t.Context())

		const submitters = 10
		const jobsPerSubmitter = 50
		totalJobs := submitters * jobsPerSubmitter

		var completed atomic.Int64

		go func() {
			for r := range p.Results() {
				if r.Err != nil {
					t.Errorf("job %d error: %v", r.JobID, r.Err)
				}
				completed.Add(1)
			}
		}()

		var submitWg sync.WaitGroup
		for s := 0; s < submitters; s++ {
			submitWg.Add(1)
			go func(submitterID int) {
				defer submitWg.Done()
				for j := 0; j < jobsPerSubmitter; j++ {
					id := submitterID*jobsPerSubmitter + j
					_ = p.Submit(ctx, Job{
						ID: id,
						Task: func(ctx context.Context) error {
							return nil
						},
					})
				}
			}(s)
		}

		submitWg.Wait()
		p.Stop()
		synctest.Wait()

		if completed.Load() != int64(totalJobs) {
			t.Fatalf("expected %d completed, got %d", totalJobs, completed.Load())
		}
	})
}

func TestPool_RateLimiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Allow 50 jobs/sec, burst 5.
		p := New(10, rate.Limit(50), 5)
		ctx := p.Start(t.Context())

		const jobCount = 30
		start := time.Now()

		go func() {
			for range p.Results() {
			}
		}()

		for i := 0; i < jobCount; i++ {
			_ = p.Submit(ctx, Job{
				ID: i,
				Task: func(ctx context.Context) error {
					return nil
				},
			})
		}

		p.Stop()
		synctest.Wait()
		elapsed := time.Since(start)

		// With 30 jobs at 50/sec and burst 5, minimum time ≈ (30-5)/50 = 0.5s.
		if elapsed < 400*time.Millisecond {
			t.Errorf("rate limiter not effective: %d jobs finished in %v (too fast)", jobCount, elapsed)
		}

		if p.Completed() != jobCount {
			t.Errorf("expected %d completed, got %d", jobCount, p.Completed())
		}
	})
}

func TestPool_ContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(2, rate.Limit(5), 1)
		ctx := p.Start(t.Context())

		var errCount atomic.Int64
		go func() {
			for r := range p.Results() {
				if r.Err != nil {
					errCount.Add(1)
				}
			}
		}()

		var submitDone sync.WaitGroup
		submitDone.Add(1)
		go func() {
			defer submitDone.Done()
			for i := 0; i < 50; i++ {
				if err := p.Submit(ctx, Job{
					ID: i,
					Task: func(ctx context.Context) error {
						time.Sleep(time.Millisecond)
						return nil
					},
				}); err != nil {
					break
				}
			}
		}()

		// Let a few jobs start, then cancel.
		time.Sleep(100 * time.Millisecond)
		p.Cancel()
		submitDone.Wait()
		p.Stop()
		synctest.Wait()

		t.Logf("completed: %d, context errors: %d", p.Completed(), errCount.Load())
	})
}

func TestPool_ErrorPropagation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(4, rate.Inf, 1)
		ctx := p.Start(t.Context())

		expectedErr := fmt.Errorf("intentional failure")

		var errCount atomic.Int64
		go func() {
			for r := range p.Results() {
				if r.Err != nil {
					errCount.Add(1)
				}
			}
		}()

		for i := 0; i < 10; i++ {
			id := i
			_ = p.Submit(ctx, Job{
				ID: id,
				Task: func(ctx context.Context) error {
					if id%2 == 0 {
						return expectedErr
					}
					return nil
				},
			})
		}

		p.Stop()
		synctest.Wait()

		if errCount.Load() != 5 {
			t.Fatalf("expected 5 errors, got %d", errCount.Load())
		}
	})
}

func TestPool_ZeroJobs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(4, rate.Inf, 1)
		_ = p.Start(t.Context())

		go func() {
			for range p.Results() {
			}
		}()

		p.Stop()
		synctest.Wait()

		if p.Completed() != 0 {
			t.Fatalf("expected 0 completed, got %d", p.Completed())
		}
	})
}
