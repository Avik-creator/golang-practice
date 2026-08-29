package main

import (
	"fmt"

	crawler "avikmukherjee.com/m"
)

type demoFetcher map[string][]string

func (f demoFetcher) Fetch(url string) (string, []string, error) {
	urls, ok := f[url]
	if !ok {
		fmt.Printf("error: %s not found\n", url)
		return "", nil, fmt.Errorf("not found: %s", url)
	}
	fmt.Printf("fetched %s -> %v\n", url, urls)
	return "ok", urls, nil
}

func main() {
	graph := demoFetcher{
		"https://a.com": {"https://b.com", "https://c.com"},
		"https://b.com": {"https://a.com", "https://d.com"},
		"https://c.com": {"https://d.com"},
		"https://d.com": {},
	}

	fmt.Println("crawling https://a.com with concurrency 2")
	crawler.Crawl("https://a.com", 2, graph)
	fmt.Println("done")
}
