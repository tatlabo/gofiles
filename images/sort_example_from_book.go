package main_images

import (
	"fmt"
	"sort"
)

type Organ struct {
	Name   string
	Weight int
}

type SomeSlice []Organ

func (s SomeSlice) Len() int {
	n := len(s)
	return n
}
func (s SomeSlice) Less(i, j int) bool {
	return s[i].Weight < s[j].Weight
}
func (s SomeSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ByWeight struct{ SomeSlice }
type ByName struct{ SomeSlice }

func (s ByName) Less(i, j int) bool {
	return s.SomeSlice[i].Name < s.SomeSlice[j].Name
}

func (s ByWeight) Less(i, j int) bool {
	return s.SomeSlice[i].Weight < s.SomeSlice[j].Weight
}

func main_images() {
	fruits := SomeSlice{{Name: "peach", Weight: 1000}, {Name: "banana", Weight: 100}, {Name: "kiwi", Weight: 10}}
	sort.Sort(ByWeight{fruits})
	fmt.Println("Before sorting:", fruits)
	sort.Sort(ByName{fruits})
	fmt.Println("After sorting:", fruits)
}
