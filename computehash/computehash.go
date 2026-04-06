package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type pair struct {
	hash, path string
}

type fileslist []string
type result map[string]fileslist

func hasFile(path string) pair {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	hash := md5.New() // hash, in next step copy to hash, hash will be writer

	if _, err := io.Copy(hash, file); err != nil {
		log.Fatal(err)
	}

	return pair{fmt.Sprintf("%x", hash.Sum(nil)), path}
}

func searchTree(dir string) (result, error) {
	hashes := make(result)

	walker := func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.Mode().IsRegular() && fi.Size() > 0 {
			pair := hasFile(p)
			hashes[pair.hash] = append(hashes[pair.hash], pair.path)
		}
		return nil
	}

	err := filepath.Walk(dir, walker)
	if err != nil {
		return nil, err
	}
	return hashes, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s root-directory\n", os.Args[0])
		os.Exit(1)
	}

	if hashes, err := searchTree(os.Args[1]); err != nil {
		log.Fatal(err)
	} else {
		for hash, files := range hashes {
			if len(files) > 1 {
				fmt.Printf("Hash: %s\n", hash)
				for _, file := range files {
					fmt.Printf("\t%s\n", file)
				}
			}
		}
	}
}
