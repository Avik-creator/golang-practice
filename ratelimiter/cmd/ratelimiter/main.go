package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Avik-creator/golang-practice/ratelimiter"
)

func main() {
	l := ratelimiter.New(2, 2)
	fmt.Println("rate=2/sec  burst=2")

	for i := range 4 {
		fmt.Printf("Allow #%d -> %v\n", i+1, l.Allow())
	}

	fmt.Println("bucket empty; Wait for 1 token...")
	start := time.Now()
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Second, errors.New("wait hung"))
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		fmt.Printf("Wait error: %v\n", err)
		return
	}
	fmt.Printf("Wait ok after %s\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("Allow -> %v (bucket empty again)\n", l.Allow())
}
