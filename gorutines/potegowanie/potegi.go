package main

import (
	"fmt"
	"time"
)

func counter(out chan<- int) {
	for x := range 14 {
		out <- x
		time.Sleep(100 * time.Millisecond)
	}
	close(out)
}

func square(in <-chan int, out chan<- int) {
	for x := range in {
		out <- x * x
	}
	close(out)
}

func printer(in <-chan int) {
	for val := range in {
		fmt.Printf("%d\n", val)
	}
}

func main() {
	naturals := make(chan int)
	sq := make(chan int)

	go counter(naturals)

	go square(naturals, sq)

	printer(sq)
}
