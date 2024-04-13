package main

import (
	"fmt"
	"log"
)

// Fetch returns the body of URL and a slice or URLS found on that page
type Fetcher interface {
	Fetch(url string) (string, []string, error)
}

// Crawl uses fetcher to recursively crawl pages starting with url,
// to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TOOD: fetch URLs in parallel: -> channels or shared variables
	// TODO: Dont fetch the same URL: -> `records` of previous urls
	if depth <= 0 {
		return
	}

	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		log.Print(err)
		return
	}

	fmt.Printf("found: %s, %q\n", url, body)
	for _, u := range urls {
		// we should check against our records before crawling a URL
		Crawl(u, depth-1, fetcher)
	}
	return
}

func main() {
	url := "https://golang.org/"
	Crawl(url, 4, fetcher)
}

type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	return "", nil, fmt.Errorf("not found: %s", url)
}

var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
