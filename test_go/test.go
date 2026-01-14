package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// https://victoriametrics.com/blog/go-sync-mutex/index.html

func main() {

	begin := make(chan any)
	var wg sync.WaitGroup
	for i := range 5 {

		simple := func(i int) {
			<-begin
			fmt.Printf("%v has begun\n", i)
		}

		wg.Go(func() {
			simple(i)
		})
	}
	fmt.Println("Unblocking goroutines...")
	close(begin)
	wg.Wait()

	var counter atomic.Uint64
	for range 1000 {
		go func() { counter.Add(1) }()
	}
	time.Sleep(10 * time.Millisecond)
	fmt.Println(counter.Load())

	var counterTwo uint64
	for range 1000 {
		go func() { counterTwo++ }()
	}
	time.Sleep(2 * time.Millisecond)
	fmt.Println(counterTwo)

	var once sync.Once
	var count int
	task := func() {
		count++
		fmt.Printf("Executed and count %v\n", count)
	}

	for range 100 {
		go func() {
			once.Do(task)
		}()
	}

	time.Sleep(1 * time.Second)

	syncMapExample()
}

func syncMapExample() {

	var m sync.Map
	for i := range 10 {
		go func(j int) {
			m.Store(j, fmt.Sprintf("value %v", j))
		}(i)
	}

	time.Sleep(1 * time.Second)

	m.Range(func(key, value any) bool {
		fmt.Printf("Key: %v, Value: %v\n", key, value)
		return true
	})

}
