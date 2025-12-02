package main

import (
	"fmt"
	"os"
	"runtime"
)

func counter(out chan<- int) {
	for x := 0; x < 14; x++ {
		out <- x
	}
	close(out)
}

func main() {
	naturals := make(chan int)
	sq := make(chan int)

	go counter(naturals)

	go func() {
		for y := range naturals {
			fmt.Printf("%d (goroutines: %d)\n", y, runtime.NumGoroutine())
			sq <- y * y
		}

		close(sq)

	}()

	for {
		x, ok := <-sq
		if ok {
			fmt.Printf("%d\n", x)
		} else if !ok {
			os.Exit(0)
		}
	}

}
