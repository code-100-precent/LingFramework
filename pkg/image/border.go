package image

import (
	"image"
	"image/color"
	"strconv"
	"strings"
)

// AddBorder adds a border to an image
func (p *Processor) AddBorder(img image.Image, width int, colorStr string) image.Image {
	bounds := img.Bounds()
	newWidth := bounds.Dx() + width*2
	newHeight := bounds.Dy() + width*2
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Parse color (simplified: supports #RRGGBB format)
	borderColor := parseColor(colorStr)

	// Fill border color
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			if x < width || x >= newWidth-width || y < width || y >= newHeight-width {
				newImg.Set(x, y, borderColor)
			} else {
				newImg.Set(x, y, img.At(x-width, y-width))
			}
		}
	}

	return newImg
}

// parseColor parses color string (supports #RRGGBB format)
func parseColor(colorStr string) color.RGBA {
	if strings.HasPrefix(colorStr, "#") {
		colorStr = colorStr[1:]
	}

	if len(colorStr) == 6 {
		r, _ := strconv.ParseUint(colorStr[0:2], 16, 8)
		g, _ := strconv.ParseUint(colorStr[2:4], 16, 8)
		b, _ := strconv.ParseUint(colorStr[4:6], 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}

	// Default black
	return color.RGBA{0, 0, 0, 255}
}

// AddRoundCorners adds rounded corners to an image
func (p *Processor) AddRoundCorners(img image.Image, radius int) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	width := bounds.Dx()
	height := bounds.Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			inCorner := false

			// Top-left corner
			if x < radius && y < radius {
				dx := float64(radius - x)
				dy := float64(radius - y)
				if dx*dx+dy*dy > float64(radius*radius) {
					inCorner = true
				}
			}

			// Top-right corner
			if x >= width-radius && y < radius {
				dx := float64(x - (width - radius))
				dy := float64(radius - y)
				if dx*dx+dy*dy > float64(radius*radius) {
					inCorner = true
				}
			}

			// Bottom-left corner
			if x < radius && y >= height-radius {
				dx := float64(radius - x)
				dy := float64(y - (height - radius))
				if dx*dx+dy*dy > float64(radius*radius) {
					inCorner = true
				}
			}

			// Bottom-right corner
			if x >= width-radius && y >= height-radius {
				dx := float64(x - (width - radius))
				dy := float64(y - (height - radius))
				if dx*dx+dy*dy > float64(radius*radius) {
					inCorner = true
				}
			}

			if inCorner {
				newImg.Set(x, y, color.RGBA{0, 0, 0, 0}) // Transparent
			} else {
				newImg.Set(x, y, img.At(x, y))
			}
		}
	}

	return newImg
}
