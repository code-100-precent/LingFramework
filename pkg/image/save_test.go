package image

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveImage_JPEG(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.jpg")

	err := SaveImage(img, path, FormatJPEG)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	// Clean up
	os.Remove(path)
}

func TestSaveImage_PNG(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.png")

	err := SaveImage(img, path, FormatPNG)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.Remove(path)
}

func TestSaveImage_BMP(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.bmp")

	err := SaveImage(img, path, FormatBMP)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.Remove(path)
}

func TestSaveImage_TIFF(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tiff")

	err := SaveImage(img, path, FormatTIFF)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.Remove(path)
}

func TestSaveImage_WEBP(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.webp")

	err := SaveImage(img, path, FormatWEBP)
	assert.NoError(t, err)
	assert.FileExists(t, path) // Should save as JPEG

	os.Remove(path)
}

func TestSaveImage_Default(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.unknown")

	err := SaveImage(img, path, Format("unknown"))
	assert.NoError(t, err) // Should default to JPEG
	assert.FileExists(t, path)

	os.Remove(path)
}

func TestSaveImageWithQuality_JPEG(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.jpg")

	err := SaveImageWithQuality(img, path, FormatJPEG, 80)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.Remove(path)
}

func TestSaveImageWithQuality_PNG(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.png")

	err := SaveImageWithQuality(img, path, FormatPNG, 80)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.Remove(path)
}

func TestSaveImage_CreatesDirectory(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "test.jpg")

	err := SaveImage(img, path, FormatJPEG)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.RemoveAll(filepath.Join(tmpDir, "subdir"))
}

func TestSaveAsSVG(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.svg")

	err := SaveAsSVG(img, path)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	// Verify it's valid SVG
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "<svg")
	assert.Contains(t, string(data), "width=\"100\"")
	assert.Contains(t, string(data), "height=\"100\"")

	os.Remove(path)
}

func TestSaveAsSVG_CreatesDirectory(t *testing.T) {
	img := createTestImage(100, 100)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "test.svg")

	err := SaveAsSVG(img, path)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	os.RemoveAll(filepath.Join(tmpDir, "subdir"))
}

func TestSaveAsSVG_DifferentSizes(t *testing.T) {
	img := createTestImage(50, 200)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.svg")

	err := SaveAsSVG(img, path)
	assert.NoError(t, err)

	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "width=\"50\"")
	assert.Contains(t, string(data), "height=\"200\"")

	os.Remove(path)
}
