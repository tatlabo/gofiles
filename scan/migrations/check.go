package main

import (
	"fmt"
)


func main() {
	a := []int{ 0, 0}
	a = append(a, 100)
	b := a[:len(a):len(a)]
	a = append(a, 2)
	b = append(b, 3)

	fmt.Printf("%v\n", a[2])
}