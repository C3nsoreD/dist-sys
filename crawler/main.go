package main

func main() {
	url := "https://golang.org/"
	Crawl(url, 4, fetcher)
}
