package image

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessor_AddBorder(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	bordered := p.AddBorder(img, 10, "#FF0000")

	assert.NotNil(t, bordered)
	assert.Equal(t, 120, bordered.Bounds().Dx()) // 100 + 10*2
	assert.Equal(t, 120, bordered.Bounds().Dy())
}

func TestProcessor_AddBorder_Black(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	bordered := p.AddBorder(img, 5, "#000000")

	assert.NotNil(t, bordered)
}

func TestProcessor_AddBorder_White(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	bordered := p.AddBorder(img, 20, "#FFFFFF")

	assert.NotNil(t, bordered)
}

func TestProcessor_AddBorder_WithoutHash(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	bordered := p.AddBorder(img, 10, "FF0000")

	assert.NotNil(t, bordered)
}

func TestProcessor_AddBorder_InvalidColor(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	bordered := p.AddBorder(img, 10, "invalid")

	assert.NotNil(t, bordered) // Should default to black
}

func TestProcessor_AddBorder_ShortColor(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	bordered := p.AddBorder(img, 10, "FF")

	assert.NotNil(t, bordered) // Should default to black
}

func TestProcessor_AddRoundCorners(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	rounded := p.AddRoundCorners(img, 20)

	assert.NotNil(t, rounded)
	assert.Equal(t, img.Bounds(), rounded.Bounds())
}

func TestProcessor_AddRoundCorners_LargeRadius(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	rounded := p.AddRoundCorners(img, 60) // Larger than half size

	assert.NotNil(t, rounded)
}

func TestProcessor_AddRoundCorners_SmallRadius(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	rounded := p.AddRoundCorners(img, 5)

	assert.NotNil(t, rounded)
}

func TestProcessor_AddRoundCorners_ZeroRadius(t *testing.T) {
	p := NewProcessor()
	img := createTestImage(100, 100)
	rounded := p.AddRoundCorners(img, 0)

	assert.NotNil(t, rounded) // Should not crash
}

func TestParseColor(t *testing.T) {
	c := parseColor("#FF0000")
	assert.Equal(t, uint8(255), c.R)
	assert.Equal(t, uint8(0), c.G)
	assert.Equal(t, uint8(0), c.B)
	assert.Equal(t, uint8(255), c.A)
}

func TestParseColor_WithoutHash(t *testing.T) {
	c := parseColor("00FF00")
	assert.Equal(t, uint8(0), c.R)
	assert.Equal(t, uint8(255), c.G)
	assert.Equal(t, uint8(0), c.B)
}

func TestParseColor_Invalid(t *testing.T) {
	c := parseColor("invalid")
	assert.Equal(t, uint8(0), c.R)
	assert.Equal(t, uint8(0), c.G)
	assert.Equal(t, uint8(0), c.B) // Default to black
}

func TestParseColor_Short(t *testing.T) {
	c := parseColor("FF")
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, c) // Default to black
}
