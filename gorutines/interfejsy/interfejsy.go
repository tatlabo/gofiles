package main

import "time"

func main() {
	chans := []chan int{make(chan int, 5), make(chan int, 5)}

	for i := range chans {
		go func(i int, ch chan<- int) {
			for {
				time.Sleep(time.Duration(i) * 500 * time.Millisecond)
				ch <- i
			}

		}(i+1, chans[i])
	}

	for j := 0; j < 10; j++ {
		select {
		case msg1 := <-chans[0]:
			println("Odebrano z kanału 1:", msg1)
		case msg2 := <-chans[1]:
			println("Odebrano z kanału 2:", msg2)

		}
	}
}
