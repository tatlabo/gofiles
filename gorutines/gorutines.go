package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func mainInGor() {
	start := time.Now()
	ch := make(chan string)

	urls := os.Args[1:]

	fn := []string{}
	for i := range urls {
		s := strings.ReplaceAll(urls[i], "/", "_")
		s = strings.ReplaceAll(s, ":", "=")

		fn = append(fn, s)
	}

	for i := range urls {
		fileName, err := os.Create(fn[i] + ".html")
		if err != nil {
			fmt.Printf("Error creating file: %v", err)
			return
		}
		defer fileName.Close()
		go fetch(urls[i], ch, fileName.Name())
	}

	for range os.Args[1:] {
		fmt.Println(<-ch)
	}

	fmt.Printf("Upłynęło %.2f\n", time.Since(start).Seconds())
}

func fetch(url string, ch chan<- string, fileName string) {
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

	if err != nil {
		ch <- fmt.Sprintf("Error creating file: %v", err)
		return
	}

	nbytes, err := io.Copy(io.Discard, resp.Body)

	secs := time.Since(start).Seconds()

	ch <- fmt.Sprintf("Fetched %s, size %d bytes in %v", url, nbytes, secs)
}
