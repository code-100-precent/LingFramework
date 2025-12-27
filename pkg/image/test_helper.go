package image

import (
	"image"
	"image/color"
)

// createTestImage creates a test image with the specified dimensions
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / width), uint8(y * 255 / height), 128, 255})
		}
	}
	return img
}
