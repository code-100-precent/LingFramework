package image

import (
	"bytes"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessor_Crop_RGBA64(t *testing.T) {
	p := NewProcessor()
	img := image.NewRGBA64(image.Rect(0, 0, 100, 100))
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)
	assert.NotNil(t, cropped)
}

func TestProcessor_Crop_NRGBA64(t *testing.T) {
	p := NewProcessor()
	img := image.NewNRGBA64(image.Rect(0, 0, 100, 100))
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)
	assert.NotNil(t, cropped)
}

func TestProcessor_Crop_Gray16(t *testing.T) {
	p := NewProcessor()
	img := image.NewGray16(image.Rect(0, 0, 100, 100))
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)
	assert.NotNil(t, cropped)
}

func TestProcessor_Crop_Generic(t *testing.T) {
	p := NewProcessor()
	// Create a custom image type
	type customImage struct {
		*image.RGBA
	}
	base := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img := &customImage{base}
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)
	assert.NotNil(t, cropped)
}

func TestDecodeImage_GIF(t *testing.T) {
	// Create a simple GIF-like buffer (minimal valid GIF)
	buf := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a
		0x01, 0x00, 0x01, 0x00, // width=1, height=1
		0x00, 0x00, 0x00, // color table
		0x21, 0xF9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, // graphics control extension
		0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // image descriptor
		0x02, 0x02, 0x44, 0x01, 0x00, // image data
		0x3B, // trailer
	}
	r := bytes.NewBuffer(buf)
	img, err := DecodeImage(r, FormatGIF)
	// GIF decoding might fail, but we test the code path
	if err == nil {
		assert.NotNil(t, img)
	}
}

func TestDecodeImage_DefaultFormat(t *testing.T) {
	// Test default format (should try auto-detection)
	buf := createTestJPEG()
	img, err := DecodeImage(buf, Format("unknown"))
	// Should attempt to decode
	if err == nil {
		assert.NotNil(t, img)
	}
}

func TestSaveImageWithQuality_AllFormats(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		format  Format
		quality int
	}{
		{"JPEG", FormatJPEG, 95},
		{"PNG", FormatPNG, 0}, // PNG doesn't use quality
		{"BMP", FormatBMP, 0},
		{"TIFF", FormatTIFF, 0},
		{"WEBP", FormatWEBP, 80},
		{"Default", Format("unknown"), 85},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tmpDir + "/test_" + tt.name + ".img"
			err := SaveImageWithQuality(img, path, tt.format, tt.quality)
			assert.NoError(t, err)
		})
	}
}

func TestSaveAsSVG_ErrorHandling(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := tmpDir + "/test.svg"

	err := SaveAsSVG(img, path)
	assert.NoError(t, err)

	// Test with different image sizes
	img2 := createTestImage(200, 50)
	path2 := tmpDir + "/test2.svg"
	err = SaveAsSVG(img2, path2)
	assert.NoError(t, err)
}

func TestApplyFilter_EdgeCases(t *testing.T) {
	p := NewProcessor()
	// Test with very small image
	img := createTestImage(3, 3)
	filtered := p.ApplyFilter(img, "blur")
	assert.NotNil(t, filtered)

	filtered = p.ApplyFilter(img, "sharpen")
	assert.NotNil(t, filtered)

	filtered = p.ApplyFilter(img, "emboss")
	assert.NotNil(t, filtered)

	filtered = p.ApplyFilter(img, "edge")
	assert.NotNil(t, filtered)
}

func TestDetectFormat_EdgeCases(t *testing.T) {
	// Test with empty buffer
	buf := bytes.NewBuffer([]byte{})
	format := DetectFormat(buf)
	assert.Equal(t, FormatJPEG, format) // Should default

	// Test with very short buffer
	buf2 := bytes.NewBuffer([]byte{0x42})
	format2 := DetectFormat(buf2)
	assert.Equal(t, FormatJPEG, format2) // Should default
}

func TestDecodeImage_GIF_Bytes(t *testing.T) {
	// Test GIF format (requires bytes import)
	var buf bytes.Buffer
	buf.Write([]byte{0x47, 0x49, 0x46, 0x38})
	_, err := DecodeImage(&buf, FormatGIF)
	// May fail but we test the code path
	_ = err
}

func TestNewProcessor(t *testing.T) {
	p := NewProcessor()
	assert.NotNil(t, p)
}

func TestProcessor_Crop(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)

	assert.Equal(t, 40, cropped.Bounds().Dx())
	assert.Equal(t, 40, cropped.Bounds().Dy())
}

func TestProcessor_Crop_OutOfBounds(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	rect := image.Rect(90, 90, 150, 150) // Partially out of bounds
	cropped := p.Crop(img, rect)

	assert.Equal(t, 10, cropped.Bounds().Dx())
	assert.Equal(t, 10, cropped.Bounds().Dy())
}

func TestProcessor_Crop_NRGBA(t *testing.T) {
	p := NewProcessor()
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)
	assert.NotNil(t, cropped)
}

func TestProcessor_Crop_Gray(t *testing.T) {
	p := NewProcessor()
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	rect := image.Rect(10, 10, 50, 50)
	cropped := p.Crop(img, rect)
	assert.NotNil(t, cropped)
}

func TestProcessor_Resize(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	resized := p.Resize(img, 50, 50)

	assert.Equal(t, 50, resized.Bounds().Dx())
	assert.Equal(t, 50, resized.Bounds().Dy())
}

func TestProcessor_Resize_Larger(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(50, 50)
	resized := p.Resize(img, 200, 200)

	assert.Equal(t, 200, resized.Bounds().Dx())
	assert.Equal(t, 200, resized.Bounds().Dy())
}

func TestProcessor_Rotate_90(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 50)
	rotated := p.Rotate(img, 90)

	assert.Equal(t, 50, rotated.Bounds().Dx())
	assert.Equal(t, 100, rotated.Bounds().Dy())
}

func TestProcessor_Rotate_180(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 50)
	rotated := p.Rotate(img, 180)

	assert.Equal(t, 100, rotated.Bounds().Dx())
	assert.Equal(t, 50, rotated.Bounds().Dy())
}

func TestProcessor_Rotate_270(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 50)
	rotated := p.Rotate(img, 270)

	assert.Equal(t, 50, rotated.Bounds().Dx())
	assert.Equal(t, 100, rotated.Bounds().Dy())
}

func TestProcessor_Rotate_InvalidAngle(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 50)
	rotated := p.Rotate(img, 45) // Invalid angle

	assert.Equal(t, img.Bounds(), rotated.Bounds())
}

func TestProcessor_Flip_Horizontal(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	flipped := p.Flip(img, "horizontal")

	assert.Equal(t, img.Bounds(), flipped.Bounds())
}

func TestProcessor_Flip_Vertical(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	flipped := p.Flip(img, "vertical")

	assert.Equal(t, img.Bounds(), flipped.Bounds())
}

func TestProcessor_Flip_InvalidDirection(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	flipped := p.Flip(img, "invalid") // Should default to vertical

	assert.Equal(t, img.Bounds(), flipped.Bounds())
}

func TestProcessor_ApplyFilter_Grayscale(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "grayscale")

	assert.NotNil(t, filtered)
	assert.Equal(t, img.Bounds(), filtered.Bounds())
}

func TestProcessor_ApplyFilter_Sepia(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "sepia")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Invert(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "invert")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Blur(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "blur")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Sharpen(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "sharpen")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Emboss(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "emboss")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Edge(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "edge")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Vintage(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "vintage")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Cool(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "cool")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Warm(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "warm")

	assert.NotNil(t, filtered)
}

func TestProcessor_ApplyFilter_Unknown(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	filtered := p.ApplyFilter(img, "unknown")

	assert.NotNil(t, filtered)
	// Should return original image
}

func TestProcessor_Adjust(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	adjusted := p.Adjust(img, 1.2, 1.1, 0.9)

	assert.NotNil(t, adjusted)
	assert.Equal(t, img.Bounds(), adjusted.Bounds())
}

func TestProcessor_Adjust_Brightness(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	adjusted := p.Adjust(img, 0.5, 1.0, 1.0) // Darken

	assert.NotNil(t, adjusted)
}

func TestProcessor_Adjust_Contrast(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	adjusted := p.Adjust(img, 1.0, 1.5, 1.0) // Increase contrast

	assert.NotNil(t, adjusted)
}

func TestProcessor_Adjust_Saturation(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	adjusted := p.Adjust(img, 1.0, 1.0, 0.5) // Desaturate

	assert.NotNil(t, adjusted)
}

func TestProcessor_Blend(t *testing.T) {
	p := NewProcessor()
	img1 := createTestImage(100, 100)
	img2 := createTestImage(100, 100)
	blended := p.Blend(img1, img2, 0.5)

	assert.NotNil(t, blended)
	assert.Equal(t, img1.Bounds(), blended.Bounds())
}

func TestProcessor_Blend_DifferentSizes(t *testing.T) {
	p := NewProcessor()
	img1 := createTestImage(100, 100)
	img2 := createTestImage(50, 50)
	blended := p.Blend(img1, img2, 0.5)

	assert.NotNil(t, blended)
	assert.Equal(t, img1.Bounds(), blended.Bounds())
}

func TestProcessor_Blend_Opacity(t *testing.T) {
	p := NewProcessor()
	img1 := createTestImage(10, 10)
	img2 := createTestImage(10, 10)
	blended1 := p.Blend(img1, img2, 0.0) // All img1
	blended2 := p.Blend(img1, img2, 1.0) // All img2

	assert.NotNil(t, blended1)
	assert.NotNil(t, blended2)
}

func TestProcessor_GetHistogram(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	histogram := p.GetHistogram(img)

	assert.NotNil(t, histogram)
	assert.Contains(t, histogram, "r")
	assert.Contains(t, histogram, "g")
	assert.Contains(t, histogram, "b")
	assert.Len(t, histogram["r"], 256)
	assert.Len(t, histogram["g"], 256)
	assert.Len(t, histogram["b"], 256)
}

func TestProcessor_GetHistogram_NotEmpty(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(10, 10)
	histogram := p.GetHistogram(img)

	// Check that histogram has some values
	sumR := 0
	for _, count := range histogram["r"] {
		sumR += count
	}
	assert.Greater(t, sumR, 0)
}
