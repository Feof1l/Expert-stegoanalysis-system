package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

func LoadPixels(path string) ([]uint8, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	pixels := make([]uint8, 0, bounds.Dx()*bounds.Dy()*3)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixels = append(pixels, uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}
	}
	return pixels, nil
}
