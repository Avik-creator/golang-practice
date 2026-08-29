package ratelimiter

import (
	"context"
	"sync"
	"time"
)

type Limiter struct {
	mu     sync.Mutex
	rate   float64 // tokens added per second
	burst  float64 // max tokens the bucket can hold
	tokens float64
	last   time.Time
	now    func() time.Time
}

func New(rate float64, burst int) *Limiter {
	if rate <= 0 || burst < 1 {
		panic("rate and burst must be positive")
	}
	clock := time.Now
	return &Limiter{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst), // start full so the first `burst` calls succeed
		last:   clock(),
		now:    clock,
	}
}

// refillLocked adds tokens earned since last refill, then caps at burst.
// Caller must hold l.mu. No background goroutine — time is applied on demand.
func (l *Limiter) refillLocked() {
	now := l.now()
	elapsed := now.Sub(l.last).Seconds()
	if elapsed <= 0 {
		return
	}

	l.tokens = min(l.tokens+elapsed*l.rate, l.burst)
	l.last = now
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refillLocked()
	if l.tokens >= 1 {
		l.tokens -= 1
		return true
	}
	return false
}

func (l *Limiter) Wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		l.mu.Lock()
		l.refillLocked()

		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}

		need := 1 - l.tokens
		delay := time.Duration(need / l.rate * float64(time.Second))
		if delay < 0 {
			delay = 0
		}
		l.mu.Unlock() // never hold the mutex while sleeping

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// loop: refill and try to take one token
		}
	}
}
