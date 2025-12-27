package image

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"golang.org/x/image/bmp"

	"github.com/stretchr/testify/assert"
)

// Helper function to create test images
func createTestJPEG() *bytes.Buffer {
	buf := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	jpeg.Encode(buf, img, nil)
	return buf
}

func createTestPNG() *bytes.Buffer {
	buf := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	png.Encode(buf, img)
	return buf
}

func createTestBMP() *bytes.Buffer {
	buf := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	bmp.Encode(buf, img)
	return buf
}

func TestDetectFormat_JPEG(t *testing.T) {
	buf := createTestJPEG()
	format := DetectFormat(buf)
	assert.Equal(t, FormatJPEG, format)
}

func TestDetectFormat_PNG(t *testing.T) {
	buf := createTestPNG()
	format := DetectFormat(buf)
	assert.Equal(t, FormatPNG, format)
}

func TestDetectFormat_BMP(t *testing.T) {
	buf := createTestBMP()
	format := DetectFormat(buf)
	assert.Equal(t, FormatBMP, format)
}

func TestDetectFormat_GIF(t *testing.T) {
	// GIF signature: 47 49 46 38
	buf := bytes.NewBuffer([]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61})
	format := DetectFormat(buf)
	assert.Equal(t, FormatGIF, format)
}

func TestDetectFormat_WEBP(t *testing.T) {
	// WEBP signature: RIFF...WEBP
	buf := bytes.NewBuffer([]byte("RIFF\x00\x00\x00\x00WEBP"))
	format := DetectFormat(buf)
	assert.Equal(t, FormatWEBP, format)
}

func TestDetectFormat_TIFF(t *testing.T) {
	// TIFF little-endian signature: 49 49 2A 00
	buf := bytes.NewBuffer([]byte{0x49, 0x49, 0x2A, 0x00})
	format := DetectFormat(buf)
	assert.Equal(t, FormatTIFF, format)

	// TIFF big-endian signature: 4D 4D 00 2A
	buf2 := bytes.NewBuffer([]byte{0x4D, 0x4D, 0x00, 0x2A})
	format2 := DetectFormat(buf2)
	assert.Equal(t, FormatTIFF, format2)
}

func TestDetectFormat_Default(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x00, 0x00})
	format := DetectFormat(buf)
	assert.Equal(t, FormatJPEG, format) // Default to JPEG
}

func TestDecodeImage_JPEG(t *testing.T) {
	buf := createTestJPEG()
	img, err := DecodeImage(buf, FormatJPEG)
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestDecodeImage_PNG(t *testing.T) {
	buf := createTestPNG()
	img, err := DecodeImage(buf, FormatPNG)
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestDecodeImage_BMP(t *testing.T) {
	buf := createTestBMP()
	img, err := DecodeImage(buf, FormatBMP)
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestDecodeImage_InvalidFormat(t *testing.T) {
	buf := bytes.NewBuffer([]byte("invalid"))
	_, err := DecodeImage(buf, Format("invalid"))
	assert.Error(t, err)
}

func TestFormatFromExtension(t *testing.T) {
	assert.Equal(t, FormatJPEG, FormatFromExtension(".jpg"))
	assert.Equal(t, FormatJPEG, FormatFromExtension(".jpeg"))
	assert.Equal(t, FormatPNG, FormatFromExtension(".png"))
	assert.Equal(t, FormatGIF, FormatFromExtension(".gif"))
	assert.Equal(t, FormatWEBP, FormatFromExtension(".webp"))
	assert.Equal(t, FormatBMP, FormatFromExtension(".bmp"))
	assert.Equal(t, FormatTIFF, FormatFromExtension(".tiff"))
	assert.Equal(t, FormatTIFF, FormatFromExtension(".tif"))
	assert.Equal(t, FormatJPEG, FormatFromExtension(".unknown")) // Default
}

func TestFormatFromExtension_WithoutDot(t *testing.T) {
	assert.Equal(t, FormatJPEG, FormatFromExtension("jpg"))
	assert.Equal(t, FormatPNG, FormatFromExtension("png"))
}

func TestFormatFromExtension_CaseInsensitive(t *testing.T) {
	assert.Equal(t, FormatJPEG, FormatFromExtension(".JPG"))
	assert.Equal(t, FormatPNG, FormatFromExtension(".PNG"))
}

func TestFormatFromFilename(t *testing.T) {
	assert.Equal(t, FormatJPEG, FormatFromFilename("test.jpg"))
	assert.Equal(t, FormatPNG, FormatFromFilename("test.png"))
	assert.Equal(t, FormatGIF, FormatFromFilename("test.gif"))
	assert.Equal(t, FormatJPEG, FormatFromFilename("test")) // No extension
	assert.Equal(t, FormatJPEG, FormatFromFilename(""))     // Empty
}

func TestIsValidFormat(t *testing.T) {
	assert.True(t, IsValidFormat(FormatJPEG))
	assert.True(t, IsValidFormat(FormatPNG))
	assert.True(t, IsValidFormat(FormatGIF))
	assert.True(t, IsValidFormat(FormatWEBP))
	assert.True(t, IsValidFormat(FormatBMP))
	assert.True(t, IsValidFormat(FormatTIFF))
	assert.False(t, IsValidFormat(Format("invalid")))
	assert.False(t, IsValidFormat(Format("")))
}
