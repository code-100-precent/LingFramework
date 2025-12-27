package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessor_AddWatermark_TopLeft(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "Test", "top-left")

	assert.NotNil(t, watermarked)
	assert.Equal(t, img.Bounds(), watermarked.Bounds())
}

func TestProcessor_AddWatermark_TopRight(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "Test", "top-right")

	assert.NotNil(t, watermarked)
}

func TestProcessor_AddWatermark_BottomLeft(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "Test", "bottom-left")

	assert.NotNil(t, watermarked)
}

func TestProcessor_AddWatermark_BottomRight(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "Test", "bottom-right")

	assert.NotNil(t, watermarked)
}

func TestProcessor_AddWatermark_Center(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "Test", "center")

	assert.NotNil(t, watermarked)
}

func TestProcessor_AddWatermark_Default(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "Test", "invalid")

	assert.NotNil(t, watermarked) // Should default to bottom-right
}

func TestProcessor_AddWatermark_EmptyText(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "", "top-left")

	assert.NotNil(t, watermarked)
}

func TestProcessor_AddWatermark_LongText(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(200, 200)
	watermarked := p.AddWatermark(img, "This is a very long watermark text", "top-left")

	assert.NotNil(t, watermarked)
}
