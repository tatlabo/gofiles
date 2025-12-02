package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type pair struct {
	hash string
	path string
}

type fileList []string
type result map[string]fileList

func hashFile(path string) pair {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		panic(err)
	}
	return pair{fmt.Sprintf("%x", hash.Sum(nil)), path}
}

func searchTree(dir string) (result, error) {
	var hashes = make(result)

	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {

		if fi.Mode().IsRegular() && fi.Size() > 0 {
			h := hashFile(p)
			hashes[h.hash] = append(hashes[h.hash], h.path)
		}

		return nil

	})

	if err != nil {
		return nil, err
	}

	return hashes, nil
}

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Missing parameter, provide dir name! ")
	}

	msg := []string{}
	if hashes, err := searchTreeConcurrent(os.Args[1]); err == nil {
		// fmt.Printf("%#v\n", hashes)
		for hash, files := range hashes {
			fname := ""
			if len(files) > 1 {
				for _, file := range files {
					fname += file + "\n"
				}
				msg = append(msg, fmt.Sprintf("%s:\n%s\n", hash, fname))
			}
		}
		writeToFile("duplikacate.txt", fmt.Sprintf("Duplicate files for hash\n%s:\n", msg))
	}

}

func writeToFile(filename string, data string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		return err
	}
	return nil
}

func collectHashes(pairs <-chan pair, results chan<- result) {
	hashes := make(result)

	for p := range pairs {
		hashes[p.hash] = append(hashes[p.hash], p.path)
	}

	results <- hashes
}

func processFiles(paths <-chan string, pairs chan<- pair, done chan<- bool) {
	for path := range paths {
		pairs <- hashFile(path)
	}

	done <- true
}

func searchTreeConcurrent(dir string) (result, error) {
	workers := 2 * runtime.GOMAXPROCS(0)
	paths := make(chan string)
	pairs := make(chan pair)
	done := make(chan bool)
	results := make(chan result)

	for range workers {
		go processFiles(paths, pairs, done)
	}

	go collectHashes(pairs, results)

	go func() {
		err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
			if fi.Mode().IsRegular() && fi.Size() > 0 {
				paths <- p
			}
			return nil
		})
		if err != nil {
			log.Printf("Error walking directory: %v", err)
		}
		close(paths)
	}()

	// Wait for all workers to finish
	for range workers {
		<-done
	}
	close(pairs)

	// Get results
	hashes := <-results
	return hashes, nil
}
