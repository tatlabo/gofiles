package main

import (
	"log"
	"time"
)

func main() {
	chans := []chan int{
		make(chan int),
		make(chan int),
	}

	for i := range chans {
		go func(ch chan int, id int) {
			for {
				time.Sleep(time.Millisecond * 100 * time.Duration(id+1))
				ch <- id
			}
		}(chans[i], i+1)
	}

	for range 12 {
		select {
		case m1 := <-chans[0]:
			log.Printf("Received from chan 1: %d", m1)
		case m2 := <-chans[1]:
			log.Printf("Received from chan 2: %d", m2)
		}
	}
}
