// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

//go:build ignore
// +build ignore

// Polecenie thumbnail tworzy miniaturki obrazów z plików JPEG,
// których nazwy są dostarczane w każdej linii standardowego wejścia.
//
// Znacznik "+build ignore" (patrz rozdział 10.) wyłącza ten plik z pakietu
// thumbnail, ale może on być skompilowany jako polecenie
// i utuchomiony w ten sposób:
//
// Uruchamiaj za pomocą:
//
//	$ go run $GOPATH/src/code/r08/thumbnail/main.go
//	foo.jpeg
//	^D
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"gofiles/gorutines/thumbnail"
)

func main() {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		thumb, err := thumbnail.ImageFile(input.Text())
		if err != nil {
			log.Print(err)
			continue
		}
		fmt.Println(thumb)
	}
	if err := input.Err(); err != nil {
		log.Fatal(err)
	}
}
