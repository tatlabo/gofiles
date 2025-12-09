package main

import (
	"net/http"
	"time"
)

type result struct {
	url     string
	err     error
	latency time.Duration
}

type urlToGet map[string]bool

func get(u string, ch chan<- result) {
	start := time.Now()

	if resp, err := http.Get(u); err != nil {
		ch <- result{url: u, err: err, latency: 0}
	} else {
		ch <- result{url: u, err: nil, latency: time.Since(start)}
		resp.Body.Close()
	}
	// close(ch)
}

func main() {

	stopper := time.After(3 * time.Second)

	urls := urlToGet{
		"https://www.fsmgov.org/":          false,
		"https://mof.gov.mh/":              false,
		"https://www.palaugov.pw/":         false,
		"https://www.nauru.gov.nr/":        false,
		"https://www.kiribati.gov.ki/":     false,
		"https://ferry.tuvalu.gov.tv/home": false,
		"https://www.samoa.travel/":        false,
		"https://www.medium.com":           false,
	}

	results := make(chan result, len(urls))

	for v := range urls {
		go get(v, results)
	}

	for range urls {
		select {
		case <-stopper:
			println("Timeout reached, exiting.")
			for u := range urls {
				if !urls[u] {
					println("Not fetched:", u)
				}
			}
			return
		case res := <-results:
			if res.err != nil {
				println("Error fetching", res.url, ":", res.err.Error())
			} else {
				urls[res.url] = true
				println("Fetched", res.url, "in", res.latency.String())
			}
		}
	}

}
