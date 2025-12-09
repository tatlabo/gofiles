package main

import (
	"io"
	"os"
	"strconv"
)

func Otput(w io.Writer, s string, params ...string) (n int, err error) {
	return w.Write([]byte(s))
}

func OtputRaw(w io.Writer, b []byte, params ...string) (n int, err error) {
	return w.Write(b)
}

func main() {
	var hello = "Hello, Write something żłów\n"
	var number = 42
	var file = "output.txt"
	f, err := os.Create(file)

	var o = os.Stdout

	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Otput(f, hello)
	// Otput(f, strconv.Itoa(number))

	OtputRaw(o, []byte(hello))
	Otput(os.Stdout, strconv.Itoa(number))
}
