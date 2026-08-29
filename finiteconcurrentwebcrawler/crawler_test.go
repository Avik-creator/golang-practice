package crawler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeFetcher map[string][]string

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	urls, ok := f[url]
	if !ok {
		return "", nil, fmt.Errorf("not found: %s", url)
	}
	return "ok", urls, nil
}

type countingFetcher struct {
	graph  fakeFetcher
	mu     sync.Mutex
	counts map[string]int
}

func (c *countingFetcher) Fetch(url string) (string, []string, error) {
	c.mu.Lock()
	c.counts[url]++
	c.mu.Unlock()
	return c.graph.Fetch(url)
}

func (c *countingFetcher) count(url string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[url]
}

type slowFetcher struct {
	graph       fakeFetcher
	inFlight    atomic.Int32
	maxInFlight atomic.Int32
}

func (s *slowFetcher) Fetch(url string) (string, []string, error) {
	n := s.inFlight.Add(1)
	for {
		old := s.maxInFlight.Load()
		if n <= old || s.maxInFlight.CompareAndSwap(old, n) {
			break
		}
	}
	defer s.inFlight.Add(-1)

	time.Sleep(20 * time.Millisecond)
	return s.graph.Fetch(url)
}

var testGraph = fakeFetcher{
	"https://a.com": {"https://b.com", "https://c.com"},
	"https://b.com": {"https://a.com", "https://d.com"},
	"https://c.com": {"https://d.com"},
	"https://d.com": {},
}

func crawlWithTimeout(t *testing.T, start string, concurrency int, f Fetcher) {
	t.Helper()
	ctx, cancel := context.WithTimeoutCause(t.Context(), 2*time.Second, errors.New("crawl hung"))
	defer cancel()

	done := make(chan struct{})
	go func() {
		Crawl(start, concurrency, f)
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal(context.Cause(ctx))
	}
}

func TestCrawl_fetchesEachURLOnce(t *testing.T) {
	f := &countingFetcher{
		graph:  testGraph,
		counts: make(map[string]int),
	}

	crawlWithTimeout(t, "https://a.com", 2, f)

	want := []string{"https://a.com", "https://b.com", "https://c.com", "https://d.com"}
	for _, u := range want {
		if got := f.count(u); got != 1 {
			t.Errorf("%s: got %d fetches, want 1", u, got)
		}
	}
	if len(f.counts) != len(want) {
		t.Errorf("fetched %d unique URLs, want %d", len(f.counts), len(want))
	}
}

func TestCrawl_diamondFetchesDOnce(t *testing.T) {
	graph := fakeFetcher{
		"https://a.com": {"https://b.com", "https://c.com"},
		"https://b.com": {"https://d.com"},
		"https://c.com": {"https://d.com"},
		"https://d.com": {},
	}
	f := &countingFetcher{
		graph:  graph,
		counts: make(map[string]int),
	}

	crawlWithTimeout(t, "https://a.com", 2, f)

	if got := f.count("https://d.com"); got != 1 {
		t.Errorf("https://d.com: got %d fetches, want 1", got)
	}
}

func TestCrawl_limitsConcurrency(t *testing.T) {
	const concurrency = 2
	links := make([]string, 5)
	graph := fakeFetcher{"https://root.com": links}
	for i := range 5 {
		u := fmt.Sprintf("https://u%d.com", i)
		links[i] = u
		graph[u] = nil
	}

	f := &slowFetcher{graph: graph}
	crawlWithTimeout(t, "https://root.com", concurrency, f)

	if got := f.maxInFlight.Load(); got > concurrency {
		t.Errorf("max in-flight fetches = %d, want <= %d", got, concurrency)
	}
	if got := f.maxInFlight.Load(); got < 2 {
		t.Errorf("max in-flight fetches = %d, want at least 2 overlapping fetches", got)
	}
}

func TestCrawl_continuesAfterFetchError(t *testing.T) {
	graph := fakeFetcher{
		"https://a.com": {"https://b.com", "https://missing.com", "https://c.com"},
		"https://b.com": {},
		"https://c.com": {},
	}
	f := &countingFetcher{
		graph:  graph,
		counts: make(map[string]int),
	}

	crawlWithTimeout(t, "https://a.com", 2, f)

	for _, u := range []string{"https://a.com", "https://b.com", "https://c.com", "https://missing.com"} {
		if got := f.count(u); got != 1 {
			t.Errorf("%s: got %d fetches, want 1", u, got)
		}
	}
}
