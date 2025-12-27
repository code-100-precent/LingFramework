package image

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/draw"
)

// Processor provides image processing functions
type Processor struct{}

// NewProcessor creates a new image processor
func NewProcessor() *Processor {
	return &Processor{}
}

// Crop crops an image to the specified rectangle
func (p *Processor) Crop(img image.Image, rect image.Rectangle) image.Image {
	bounds := img.Bounds()
	rect = rect.Intersect(bounds)

	switch img := img.(type) {
	case *image.RGBA:
		return img.SubImage(rect)
	case *image.RGBA64:
		return img.SubImage(rect)
	case *image.NRGBA:
		return img.SubImage(rect)
	case *image.NRGBA64:
		return img.SubImage(rect)
	case *image.Gray:
		return img.SubImage(rect)
	case *image.Gray16:
		return img.SubImage(rect)
	default:
		// Generic handling
		newImg := image.NewRGBA(rect)
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				newImg.Set(x-rect.Min.X, y-rect.Min.Y, img.At(x, y))
			}
		}
		return newImg
	}
}

// Resize resizes an image to the specified dimensions using high-quality scaling
func (p *Processor) Resize(img image.Image, width, height int) image.Image {
	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Use Catmull-Rom interpolation for high-quality scaling
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	return dst
}

// Rotate rotates an image by the specified angle (90, 180, or 270 degrees)
func (p *Processor) Rotate(img image.Image, angle int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var newWidth, newHeight int
	var newImg *image.RGBA

	switch angle {
	case 90:
		newWidth = height
		newHeight = width
		newImg = image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				newImg.Set(height-1-y, x, img.At(x, y))
			}
		}
	case 180:
		newWidth = width
		newHeight = height
		newImg = image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				newImg.Set(width-1-x, height-1-y, img.At(x, y))
			}
		}
	case 270:
		newWidth = height
		newHeight = width
		newImg = image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				newImg.Set(y, width-1-x, img.At(x, y))
			}
		}
	default:
		return img
	}

	return newImg
}

// Flip flips an image horizontally or vertically
func (p *Processor) Flip(img image.Image, direction string) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	newImg := image.NewRGBA(bounds)

	if direction == "horizontal" {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				newImg.Set(width-1-x, y, img.At(x, y))
			}
		}
	} else { // vertical
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				newImg.Set(x, height-1-y, img.At(x, y))
			}
		}
	}

	return newImg
}

// ApplyFilter applies a filter to an image
func (p *Processor) ApplyFilter(img image.Image, filterType string) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			var newR, newG, newB uint8

			switch filterType {
			case "grayscale":
				gray := uint8(0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8))
				newR, newG, newB = gray, gray, gray
			case "sepia":
				newR = uint8(math.Min(255, 0.393*float64(r8)+0.769*float64(g8)+0.189*float64(b8)))
				newG = uint8(math.Min(255, 0.349*float64(r8)+0.686*float64(g8)+0.168*float64(b8)))
				newB = uint8(math.Min(255, 0.272*float64(r8)+0.534*float64(g8)+0.131*float64(b8)))
			case "invert":
				newR, newG, newB = 255-r8, 255-g8, 255-b8
			case "blur":
				// Simple blur effect (3x3 average)
				var sumR, sumG, sumB, count uint32
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
							nr, ng, nb, _ := img.At(nx, ny).RGBA()
							sumR += nr >> 8
							sumG += ng >> 8
							sumB += nb >> 8
							count++
						}
					}
				}
				if count > 0 {
					newR = uint8(sumR / count)
					newG = uint8(sumG / count)
					newB = uint8(sumB / count)
				} else {
					newR, newG, newB = r8, g8, b8
				}
			case "sharpen":
				// Sharpen effect (Laplacian operator)
				var sumR, sumG, sumB int32
				kernel := [][]int32{
					{0, -1, 0},
					{-1, 5, -1},
					{0, -1, 0},
				}
				for ky := -1; ky <= 1; ky++ {
					for kx := -1; kx <= 1; kx++ {
						nx, ny := x+kx, y+ky
						if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
							nr, ng, nb, _ := img.At(nx, ny).RGBA()
							weight := kernel[ky+1][kx+1]
							sumR += int32(nr>>8) * weight
							sumG += int32(ng>>8) * weight
							sumB += int32(nb>>8) * weight
						}
					}
				}
				newR = uint8(math.Max(0, math.Min(255, float64(sumR))))
				newG = uint8(math.Max(0, math.Min(255, float64(sumG))))
				newB = uint8(math.Max(0, math.Min(255, float64(sumB))))
			case "emboss":
				// Emboss effect
				var sumR, sumG, sumB int32
				kernel := [][]int32{
					{-2, -1, 0},
					{-1, 1, 1},
					{0, 1, 2},
				}
				for ky := -1; ky <= 1; ky++ {
					for kx := -1; kx <= 1; kx++ {
						nx, ny := x+kx, y+ky
						if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
							nr, ng, nb, _ := img.At(nx, ny).RGBA()
							weight := kernel[ky+1][kx+1]
							sumR += int32(nr>>8) * weight
							sumG += int32(ng>>8) * weight
							sumB += int32(nb>>8) * weight
						}
					}
				}
				newR = uint8(math.Max(0, math.Min(255, float64(sumR+128))))
				newG = uint8(math.Max(0, math.Min(255, float64(sumG+128))))
				newB = uint8(math.Max(0, math.Min(255, float64(sumB+128))))
			case "edge":
				// Edge detection
				var sumR, sumG, sumB int32
				kernel := [][]int32{
					{-1, -1, -1},
					{-1, 8, -1},
					{-1, -1, -1},
				}
				for ky := -1; ky <= 1; ky++ {
					for kx := -1; kx <= 1; kx++ {
						nx, ny := x+kx, y+ky
						if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
							nr, ng, nb, _ := img.At(nx, ny).RGBA()
							weight := kernel[ky+1][kx+1]
							sumR += int32(nr>>8) * weight
							sumG += int32(ng>>8) * weight
							sumB += int32(nb>>8) * weight
						}
					}
				}
				gray := uint8(math.Max(0, math.Min(255, (float64(sumR)+float64(sumG)+float64(sumB))/3)))
				newR, newG, newB = gray, gray, gray
			case "vintage":
				// Vintage effect
				newR = uint8(math.Min(255, 0.393*float64(r8)+0.769*float64(g8)+0.189*float64(b8)))
				newG = uint8(math.Min(255, 0.349*float64(r8)+0.686*float64(g8)+0.168*float64(b8)))
				newB = uint8(math.Min(255, 0.272*float64(r8)+0.534*float64(g8)+0.131*float64(b8)))
				// Add some noise
				if (x+y)%3 == 0 {
					noise := uint8((x + y) % 20)
					newR = uint8(math.Min(255, float64(newR)+float64(noise)))
					newG = uint8(math.Min(255, float64(newG)+float64(noise)))
					newB = uint8(math.Min(255, float64(newB)+float64(noise)))
				}
			case "cool":
				// Cool tone
				newR = uint8(math.Min(255, float64(r8)*0.9))
				newG = uint8(math.Min(255, float64(g8)*1.0))
				newB = uint8(math.Min(255, float64(b8)*1.2))
			case "warm":
				// Warm tone
				newR = uint8(math.Min(255, float64(r8)*1.2))
				newG = uint8(math.Min(255, float64(g8)*1.1))
				newB = uint8(math.Min(255, float64(b8)*0.9))
			default:
				newR, newG, newB = r8, g8, b8
			}

			newImg.Set(x, y, color.RGBA{newR, newG, newB, uint8(a >> 8)})
		}
	}

	return newImg
}

// Adjust adjusts brightness, contrast, and saturation of an image
func (p *Processor) Adjust(img image.Image, brightness, contrast, saturation float64) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8 := float64(r >> 8)
			g8 := float64(g >> 8)
			b8 := float64(b >> 8)

			// Adjust brightness
			r8 *= brightness
			g8 *= brightness
			b8 *= brightness

			// Adjust contrast
			r8 = (r8-128)*contrast + 128
			g8 = (g8-128)*contrast + 128
			b8 = (b8-128)*contrast + 128

			// Adjust saturation
			gray := 0.299*r8 + 0.587*g8 + 0.114*b8
			r8 = gray + (r8-gray)*saturation
			g8 = gray + (g8-gray)*saturation
			b8 = gray + (b8-gray)*saturation

			// Clamp values
			r8 = math.Max(0, math.Min(255, r8))
			g8 = math.Max(0, math.Min(255, g8))
			b8 = math.Max(0, math.Min(255, b8))

			newImg.Set(x, y, color.RGBA{
				R: uint8(r8),
				G: uint8(g8),
				B: uint8(b8),
				A: uint8(a >> 8),
			})
		}
	}

	return newImg
}

// Blend blends two images with specified opacity
func (p *Processor) Blend(img1, img2 image.Image, opacity float64) image.Image {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	newImg := image.NewRGBA(bounds1)

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			r1, g1, b1, a1 := img1.At(x, y).RGBA()

			x2 := x
			y2 := y
			if x2 >= bounds2.Min.X && x2 < bounds2.Max.X && y2 >= bounds2.Min.Y && y2 < bounds2.Max.Y {
				r2, g2, b2, a2 := img2.At(x2, y2).RGBA()

				r := uint8((float64(r1>>8)*(1-opacity) + float64(r2>>8)*opacity))
				g := uint8((float64(g1>>8)*(1-opacity) + float64(g2>>8)*opacity))
				b := uint8((float64(b1>>8)*(1-opacity) + float64(b2>>8)*opacity))
				a := uint8((float64(a1>>8)*(1-opacity) + float64(a2>>8)*opacity))

				newImg.Set(x, y, color.RGBA{r, g, b, a})
			} else {
				newImg.Set(x, y, img1.At(x, y))
			}
		}
	}

	return newImg
}

// GetHistogram gets histogram information for an image
func (p *Processor) GetHistogram(img image.Image) map[string][]int {
	bounds := img.Bounds()
	histogram := map[string][]int{
		"r": make([]int, 256),
		"g": make([]int, 256),
		"b": make([]int, 256),
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			histogram["r"][r>>8]++
			histogram["g"][g>>8]++
			histogram["b"][b>>8]++
		}
	}

	return histogram
}
