// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/


// Package thumbnail generuje miniaturki z większych obrazów.
// Aktualnie obsługiwane są tylko obrazy JPEG.
package thumbnail

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Image zwraca miniaturową wersję obrazu źródłowego (src).
func Image(src image.Image) image.Image {
	// Oblicza rozmiar miniaturki, zachowując współczynnik proporcji.
	xs := src.Bounds().Size().X
	ys := src.Bounds().Size().Y
	width, height := 128, 128
	if aspect := float64(xs) / float64(ys); aspect < 1.0 {
		width = int(128 * aspect) // orientacja pionowa
	} else {
		height = int(128 / aspect) // orientacja pozioma
	}
	xscale := float64(xs) / float64(width)
	yscale := float64(ys) / float64(height)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Bardzo prymitywny algorytm skalowania.
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			srcx := int(float64(x) * xscale)
			srcy := int(float64(y) * yscale)
			dst.Set(x, y, src.At(srcx, srcy))
		}
	}
	return dst
}

// ImageStream odczytuje obraz z r
// i zapisuje jego wersję miniaturową w zmiennej w.
func ImageStream(w io.Writer, r io.Reader) error {
	src, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	dst := Image(src)
	return jpeg.Encode(w, dst, nil)
}

// ImageFile2 odczytuje obraz z infile 
// i zapisuje jego miniaturę w outfile.
func ImageFile2(outfile, infile string) (err error) {
	in, err := os.Open(infile)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(outfile)
	if err != nil {
		return err
	}

	if err := ImageStream(out, in); err != nil {
		out.Close()
		return fmt.Errorf("skalowanie %s na %s: %s", infile, outfile, err)
	}
	return out.Close()
}

// ImageFile odczytuje obraz z infile
// i zapisuje jego miniatur w tym samym katalogu.
// Zwraca wygenerowaną nazwę pliku, np. "foo.thumb.jpg".
func ImageFile(infile string) (string, error) {
	ext := filepath.Ext(infile) // np. ".jpg", ".JPEG"
	outfile := strings.TrimSuffix(infile, ext) + ".thumb" + ext
	return outfile, ImageFile2(outfile, infile)
}
