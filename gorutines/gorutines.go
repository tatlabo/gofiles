package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	start := time.Now()
	ch := make(chan string)

	for _, url := range os.Args[1:] {
		go fetch(url, ch)
	}

	for range os.Args[1:] {
		fmt.Println(<-ch)
	}

	fmt.Printf("Upłynęło %.2f\n", time.Since(start).Seconds())
}

func fetch(url string, ch chan<- string) {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		ch <- fmt.Sprintf("Error fetching %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- fmt.Sprintf("Error fetching %s: %v", url, err)
		return
	}
	nbytes, err := io.Copy(io.Discard, resp.Body)

	secs := time.Since(start).Seconds()

	ch <- fmt.Sprintf("Fetched %s, size %d bytes in %v", url, nbytes, secs)
}
