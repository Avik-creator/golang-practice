package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Job struct {
	ID      string
	Payload string
}

type Queue interface {
	Reserve() (*Job, bool)
	Complete(id string)
	Fail(id string)
}

type Handler func(ctx context.Context, job *Job) error

func Run(ctx context.Context, workers int, q Queue, handle Handler) error {
	if workers < 1 {
		workers = 1
	}
	var inFlight atomic.Int64
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			worker(ctx, q, handle, &inFlight)
		})
	}
	wg.Wait()
	return ctx.Err()
}

func worker(ctx context.Context, q Queue, handle Handler, inFlight *atomic.Int64) {
	for {
		if ctx.Err() != nil {
			return
		}

		inFlight.Add(1)
		job, ok := q.Reserve()
		if !ok {
			inFlight.Add(-1)
			if inFlight.Load() == 0 {
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}

		err := handle(ctx, job)
		if err != nil {
			q.Fail(job.ID)
		} else {
			q.Complete(job.ID)
		}
		inFlight.Add(-1)
	}
}
