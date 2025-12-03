// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Ten plik zawiera tylko przykładowy kod z książki.
// Nie uruchamia żadnego kodu z code/r08/thumbnail.

package thumbnail_test

import (
	"log"
	"os"
	"sync"

	"gofiles/gorutines/thumbnail"
)

// !+1
// makeThumbnails tworzy miniatury określonych plików obrazów.
func makeThumbnails(filenames []string) {
	for _, f := range filenames {
		if _, err := thumbnail.ImageFile(f); err != nil {
			log.Println(err)
		}
	}
}

//!-1

// !+2
// UWAGA: nieprawidłowe!
func makeThumbnails2(filenames []string) {
	for _, f := range filenames {
		go thumbnail.ImageFile(f) // UWAGA: ignorowanie błędów
	}
}

//!-2

// !+3
// makeThumbnails3 tworzy równolegle miniaturki z określonych plików obrazów.
func makeThumbnails3(filenames []string) {
	ch := make(chan struct{})
	for _, f := range filenames {
		go func(f string) {
			thumbnail.ImageFile(f) // UWAGA: ignorowanie błędów
			ch <- struct{}{}
		}(f)
	}

	// Oczekiwanie na zakończenie wszystkich funkcji goroutine.
	for range filenames {
		<-ch
	}
}

//!-3

// !+4
// makeThumbnails4 tworzy równolegle miniaturki z określonych plików obrazów.
// It returns an error if any step failed.
func makeThumbnails4(filenames []string) error {
	errors := make(chan error)

	for _, f := range filenames {
		go func(f string) {
			_, err := thumbnail.ImageFile(f)
			errors <- err
		}(f)
	}

	for range filenames {
		if err := <-errors; err != nil {
			return err // UWAGA: nieprawidłowe: wyciek funkcji goroutine!
		}
	}

	return nil
}

//!-4

// !+5
// makeThumbnails5 tworzy równolegle miniaturki z określonych plików obrazów.
// Zwraca wygenerowane nazwy plików w dowolnej kolejności
// lub błąd, jeśli jakiś etap zawiedzie.
func makeThumbnails5(filenames []string) (thumbfiles []string, err error) {
	type item struct {
		thumbfile string
		err       error
	}

	ch := make(chan item, len(filenames))
	for _, f := range filenames {
		go func(f string) {
			var it item
			it.thumbfile, it.err = thumbnail.ImageFile(f)
			ch <- it
		}(f)
	}

	for range filenames {
		it := <-ch
		if it.err != nil {
			return nil, it.err
		}
		thumbfiles = append(thumbfiles, it.thumbfile)
	}

	return thumbfiles, nil
}

//!-5

// !+6
// makeThumbnails6 tworzy miniaturki dla każdego pliku otrzymanego z kanału.
// Zwraca liczbę bajtów zajmowanych przez utworzone pliki.
func makeThumbnails6(filenames <-chan string) int64 {
	sizes := make(chan int64)
	var wg sync.WaitGroup // liczba roboczych funkcji goroutine
	for f := range filenames {
		wg.Add(1)
		// Funkcja robocza.
		go func(f string) {
			defer wg.Done()
			thumb, err := thumbnail.ImageFile(f)
			if err != nil {
				log.Println(err)
				return
			}
			info, _ := os.Stat(thumb) // można ignorować błąd
			sizes <- info.Size()
		}(f)
	}

	// Funkcja zamykania.
	go func() {
		wg.Wait()
		close(sizes)
	}()

	var total int64
	for size := range sizes {
		total += size
	}
	return total
}

//!-6
