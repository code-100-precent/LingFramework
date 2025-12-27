package image

import (
	"image"
	"image/color"

	"golang.org/x/image/draw"
)

// AddWatermark adds a text watermark to an image
func (p *Processor) AddWatermark(img image.Image, text, position string) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, bounds, img, bounds.Min, draw.Src)

	// Calculate text dimensions (estimated)
	fontSize := 36
	charWidth := fontSize * 7 / 13
	textWidth := len(text) * charWidth
	textHeight := fontSize

	// Calculate watermark position
	var x, y int
	width := bounds.Dx()
	height := bounds.Dy()
	padding := 30

	switch position {
	case "top-left":
		x, y = padding, padding+textHeight
	case "top-right":
		x, y = width-textWidth-padding, padding+textHeight
	case "bottom-left":
		x, y = padding, height-padding
	case "bottom-right":
		x, y = width-textWidth-padding, height-padding
	case "center":
		x, y = width/2-textWidth/2, height/2
	default:
		x, y = width-textWidth-padding, height-padding
	}

	// Draw background (rounded semi-transparent background)
	bgPadding := 12
	bgRect := image.Rect(x-bgPadding, y-textHeight-bgPadding, x+textWidth+bgPadding, y+bgPadding)
	radius := 6
	bgColor := color.RGBA{20, 20, 20, 180}

	for py := bgRect.Min.Y; py < bgRect.Max.Y; py++ {
		for px := bgRect.Min.X; px < bgRect.Max.X; px++ {
			if px >= 0 && px < width && py >= 0 && py < height {
				inCorner := false

				// Check if in rounded corner area
				if px < bgRect.Min.X+radius && py < bgRect.Min.Y+radius {
					dx := float64(bgRect.Min.X + radius - px)
					dy := float64(bgRect.Min.Y + radius - py)
					if dx*dx+dy*dy > float64(radius*radius) {
						inCorner = true
					}
				}
				if px >= bgRect.Max.X-radius && py < bgRect.Min.Y+radius {
					dx := float64(px - (bgRect.Max.X - radius))
					dy := float64(bgRect.Min.Y + radius - py)
					if dx*dx+dy*dy > float64(radius*radius) {
						inCorner = true
					}
				}
				if px < bgRect.Min.X+radius && py >= bgRect.Max.Y-radius {
					dx := float64(bgRect.Min.X + radius - px)
					dy := float64(py - (bgRect.Max.Y - radius))
					if dx*dx+dy*dy > float64(radius*radius) {
						inCorner = true
					}
				}
				if px >= bgRect.Max.X-radius && py >= bgRect.Max.Y-radius {
					dx := float64(px - (bgRect.Max.X - radius))
					dy := float64(py - (bgRect.Max.Y - radius))
					if dx*dx+dy*dy > float64(radius*radius) {
						inCorner = true
					}
				}

				if !inCorner {
					oldR, oldG, oldB, oldA := newImg.At(px, py).RGBA()
					alpha := float64(bgColor.A) / 255.0
					newR := uint8(float64(bgColor.R)*alpha + float64(oldR>>8)*(1-alpha))
					newG := uint8(float64(bgColor.G)*alpha + float64(oldG>>8)*(1-alpha))
					newB := uint8(float64(bgColor.B)*alpha + float64(oldB>>8)*(1-alpha))
					newImg.Set(px, py, color.RGBA{newR, newG, newB, uint8(oldA >> 8)})
				}
			}
		}
	}

	// Draw text (white, opaque)
	textColor := color.RGBA{255, 255, 255, 255}
	p.drawText(newImg, text, x, y, fontSize, textColor)

	return newImg
}
