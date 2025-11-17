package main_images

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

type Color interface {
	// RGBA returns the alpha-premultiplied red, green, blue and alpha values
	// for the color. Each value ranges within [0, 0xFFFF], but is represented
	// by a uint32 so that multiplying by a blend factor up to 0xFFFF will not
	// overflow.
	RGBA() (r, g, b, a uint32)
}

func main() {

	path := "F:/picture/picture.jpg"
	reader, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	m, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	bounds := m.Bounds()

	var histagram [16][4]int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := m.At(x, y).RGBA()
			histagram[r>>12][0]++
			histagram[g>>12][1]++
			histagram[b>>12][2]++
			histagram[a>>12][3]++
		}
	}

	fmt.Printf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
	for i, x := range histagram {
		fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
	}
}
