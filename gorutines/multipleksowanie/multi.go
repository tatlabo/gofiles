package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Odliczanie")
	tick := time.NewTicker(1 * time.Second)
	abort := make(chan struct{})

	defer tick.Stop()

	go func() {
		os.Stdin.Read(make([]byte, 1))
		abort <- struct{}{}
	}()

	for c := 10; c >= 0; c-- {

		select {
		case <-tick.C:
			fmt.Println(c)
		case <-abort:
			fmt.Println("Odliczanie przerwane!")
			return

		}
	}

}
