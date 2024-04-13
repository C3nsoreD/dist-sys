package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/html"
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

type fetchState struct {
	fetched map[string]bool
	mu      *sync.Mutex
}

func RunSerial(url string, fetcher Fetcher, store map[string]bool) {
	if store[url] {
		return
	}

	_, urls, err := fetcher.Fetch(url)
	if err != nil {
		log.Printf("failed to fetch %v", err)
		return
	}

	for _, url := range urls {
		RunSerial(url, fetcher, store)
	}
}

// Run calls our fetcher recursively, fetching urls from initial `url`
// Is concurrent safe
func Run(url string, fetcher Fetcher, fs fetchState) {
	fs.mu.Lock()
	present := fs.fetched[url]
	fs.mu.Unlock()

	if present {
		return
	}

	_, urls, err := fetcher.Fetch(url)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, u := range urls {
		wg.Add(1)

		go func(u string) {
			defer wg.Done()
			Run(url, fetcher, fs)
		}(u)
	}
}

func makeState() *fetchState {
	return &fetchState{
		fetched: make(map[string]bool),
		mu:      &sync.Mutex{},
	}
}

func worker(url string, ch chan []string, fetcher Fetcher) {
	_, urls, err := fetcher.Fetch(url)
	if err != nil {
		log.Printf("failed to get urls %v", err)
		ch <- []string{}
	} else {
		ch <- urls
	}
}

func coordinator(ch chan []string, fetcher Fetcher) {
	n := 1 // number of workers
	fetched := make(map[string]bool)

	for urls := range ch {
		for _, url := range urls {
			if fetched[url] == false {
				n++
				go worker(url, ch, fetcher)
			}
		}
		n--
		if n == 0 {
			break
		}
	}
}

// RunCh alternative implementation using channels
func RunCh(url string, fetcher Fetcher) {
	ch := make(chan []string)
	go func() {
		ch <- []string{url}
	}()
	coordinator(ch, fetcher)
}

type fetcher map[string]*result

type result struct {
	body string
	urls []string
}

func (f fetcher) Fetch(url string) (string, []string, error) {
	urls := []string{}
	res, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", nil, err
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}

	var fn func(*html.Node)
	fn = func(h *html.Node) {
		if h.Type == html.ElementNode && h.Data == "a" {
			for _, a := range h.Attr {
				if a.Key == "href" {
					urls = append(urls, a.Val)
				}
			}
		}

		for c := h.FirstChild; c != nil; c = c.NextSibling {
			fn(c)
		}
	}

	fn(doc)
	if len(urls) <= 0 {
		log.Printf("found zero urls\n")
	}

	return string(body), urls, nil
}

type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	return "", nil, fmt.Errorf("not found: %s", url)
}

var ffetcher = fakeFetcher{
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

func main() {
	url := "https://golang.org/"
	f := fetcher{}
	Crawl(url, 4, f)
}
