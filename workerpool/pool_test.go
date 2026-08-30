package workerpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_completesAllJobs(t *testing.T) {
	jobs := make([]Job, 5)
	for i := range 5 {
		jobs[i] = Job{ID: fmt.Sprintf("job-%d", i)}
	}
	q := NewMemoryQueue(jobs...)

	err := Run(t.Context(), 2, q, func(context.Context, *Job) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := len(q.Completed()); got != 5 {
		t.Fatalf("completed %d jobs, want 5", got)
	}
}

func TestRun_limitsConcurrency(t *testing.T) {
	const workers = 2
	jobs := make([]Job, 6)
	for i := range 6 {
		jobs[i] = Job{ID: fmt.Sprintf("job-%d", i)}
	}
	q := NewMemoryQueue(jobs...)

	var inFlight, max atomic.Int64
	err := Run(t.Context(), workers, q, func(context.Context, *Job) error {
		n := inFlight.Add(1)
		for {
			old := max.Load()
			if n <= old || max.CompareAndSwap(old, n) {
				break
			}
		}
		defer inFlight.Add(-1)
		time.Sleep(20 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := max.Load(); got > workers {
		t.Fatalf("max in-flight handlers = %d, want <= %d", got, workers)
	}
	if got := max.Load(); got < 2 {
		t.Fatalf("max in-flight handlers = %d, want overlap of 2", got)
	}
}

func TestRun_stopsOnCancel(t *testing.T) {
	q := NewMemoryQueue(Job{ID: "a"}, Job{ID: "b"}, Job{ID: "c"})
	ctx, cancel := context.WithCancelCause(t.Context())

	var once sync.Once
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, 2, q, func(ctx context.Context, _ *Job) error {
			once.Do(func() { close(started) })
			<-ctx.Done()
			return ctx.Err()
		})
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}
	cancel(errors.New("stop pool"))

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run: got %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestRun_retriesFailedJob(t *testing.T) {
	q := NewMemoryQueue(Job{ID: "a"})
	var attempts atomic.Int64

	err := Run(t.Context(), 2, q, func(_ context.Context, job *Job) error {
		if attempts.Add(1) == 1 {
			return errors.New("first attempt fails")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := q.Completed(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("completed %v, want [a]", q.Completed())
	}
	if attempts.Load() < 2 {
		t.Fatalf("attempts = %d, want at least 2", attempts.Load())
	}
}
