package main

func main() {
	stream := make(chan string)

	stramString := func(ch chan string) {
		// time.Sleep(1 * time.Second)
		stream <- "Hello, channel!"
		// close(ch)
	}

	go stramString(stream)
	msg := <-stream
	println(msg)

}
