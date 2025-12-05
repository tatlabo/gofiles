package main

func main() {
	type shoe uint

	const (
		_ shoe = 1 << iota
		running
		dress
		sandal
		clog
	)

	println(running, "\n", clog)

}

func merge(out chan<- int, a, b <-chan int) {

	var aClosed, bClosed bool
	for !bClosed || !aClosed {
		select {
		case v, ok := <-a:
			if !ok {
				aClosed = true
				continue
			}
			out <- v

		case v, ok := <-b:
			{
				if !ok {
					bClosed = true
					continue
				}
				out <- v
			}
		}
		close(out)
	}
}
