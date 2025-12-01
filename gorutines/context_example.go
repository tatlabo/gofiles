package main

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"time"
)

type result struct {
	url      string
	duration time.Duration
	err      error
}

func getURL(cxt context.Context, url string, ch chan<- result) {
	start := time.Now()
	ticker := time.NewTicker(1 * time.Second).C
	resp, err := http.Get(url)

	var r result

	req, _ := http.NewRequestWithContext(cxt, http.MethodGet, url, nil)

	if resp, err = http.DefaultClient.Do(req); err != nil {
		r = result{url, 0, err}
	} else {
		duration := time.Since(start)
		r = result{url, duration, nil}
		resp.Body.Close()
	}

	for {
		select {
		case ch <- r:
			return
		case <-ticker:
			log.Print("thick", r)

		}
	}

}

func main() {
	urls := []string{
		"https://www.google.com",
		"https://www.github.com",
		"https://www.stackoverflow.com",
		"https://www.reddit.com",
		"https://www.medium.com",
		"https://tenki.jp",
	}

	res, err := first(context.Background(), urls)
	if err != nil {
		log.Fatalf("Error fetching URLs: %v", err)
	}

	if res.err != nil {
		log.Printf("Error fetching %s: %v", res.url, res.err)
	} else {
		log.Printf("First response from %s took %v", res.url, res.duration)
	}

	time.Sleep(8 * time.Second)
	log.Println("Main function completed", runtime.NumGoroutine())
}

func first(ctx context.Context, urls []string) (*result, error) {
	result := make(chan result, len(urls))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, url := range urls {
		go getURL(ctx, url, result)
	}

	select {
	case r := <-result:
		return &r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
