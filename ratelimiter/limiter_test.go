package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Unix(0, 0).UTC()}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newTestLimiter(rate float64, burst int, clock *fakeClock) *Limiter {
	l := New(rate, burst)
	l.now = clock.Now
	l.last = clock.Now()
	return l
}

func TestAllow_respectsBurst(t *testing.T) {
	clock := newFakeClock()
	l := newTestLimiter(1, 2, clock)

	if !l.Allow() || !l.Allow() {
		t.Fatal("first two Allow calls should succeed (burst=2)")
	}
	if l.Allow() {
		t.Fatal("third Allow should fail with no time passed")
	}
}

func TestAllow_refillsOverTime(t *testing.T) {
	clock := newFakeClock()
	l := newTestLimiter(1, 2, clock)

	if !l.Allow() || !l.Allow() {
		t.Fatal("wanted burst of 2")
	}

	clock.Advance(time.Second)
	if !l.Allow() {
		t.Fatal("1 token/sec should allow one more call after 1s")
	}
	if l.Allow() {
		t.Fatal("bucket should be empty again")
	}
}

func TestAllow_capsAtBurst(t *testing.T) {
	clock := newFakeClock()
	l := newTestLimiter(1, 2, clock)

	if !l.Allow() || !l.Allow() {
		t.Fatal("wanted burst of 2")
	}

	clock.Advance(10 * time.Second)
	if !l.Allow() || !l.Allow() {
		t.Fatal("after a long idle, burst is still 2")
	}
	if l.Allow() {
		t.Fatal("tokens must not grow past burst")
	}
}

func TestWait_canceledContext(t *testing.T) {
	clock := newFakeClock()
	l := newTestLimiter(1, 1, clock)
	if !l.Allow() {
		t.Fatal("wanted the one burst token")
	}

	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("stopped"))

	err := l.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait: got %v, want context.Canceled", err)
	}
}

func TestWait_fullBucketReturnsImmediately(t *testing.T) {
	l := New(1, 1)
	ctx, cancel := context.WithTimeoutCause(t.Context(), time.Second, errors.New("wait hung"))
	defer cancel()

	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait on a full bucket: %v", err)
	}
}

func TestWait_unblocksAfterRefill(t *testing.T) {
	l := New(100, 1)
	if !l.Allow() {
		t.Fatal("wanted the one burst token")
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), time.Second, errors.New("wait hung"))
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("Wait after draining burst: %v", err)
	}
}

func TestAllow_concurrentRace(t *testing.T) {
	l := New(1000, 50)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				l.Allow()
			}
		})
	}
	wg.Wait()
}
