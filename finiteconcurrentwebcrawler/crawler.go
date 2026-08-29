package crawler

import "sync"

type Fetcher interface {
	Fetch(url string) (body string, urls []string, err error)
}

type crawler struct {
	fetcher Fetcher
	mu      sync.Mutex
	visited map[string]struct{}
	sem     chan struct{}
	wg      sync.WaitGroup
}

func Crawl(start string, concurrency int, f Fetcher) {
	if concurrency < 1 {
		concurrency = 1
	}
	c := &crawler{
		fetcher: f,
		visited: make(map[string]struct{}),
		sem:     make(chan struct{}, concurrency),
	}
	c.enqueue(start)
	c.wg.Wait()
}

func (c *crawler) enqueue(url string) {
	c.mu.Lock()
	if _, seen := c.visited[url]; seen {
		c.mu.Unlock()
		return
	}

	c.visited[url] = struct{}{}
	c.mu.Unlock()

	c.wg.Add(1)
	go c.worker(url)
}

func (c *crawler) worker(url string) {
	defer c.wg.Done()

	c.sem <- struct{}{}
	_, urls, err := c.fetcher.Fetch(url)
	<-c.sem

	if err != nil {
		return
	}
	for _, u := range urls {
		c.enqueue(u)
	}
}
