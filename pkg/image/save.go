package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// SaveImage saves an image to file with specified format
func SaveImage(img image.Image, path string, format Format) error {
	return SaveImageWithQuality(img, path, format, 90)
}

// SaveImageWithQuality saves an image to file with specified format and quality
func SaveImageWithQuality(img image.Image, path string, format Format, quality int) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	switch strings.ToLower(string(format)) {
	case "jpeg", "jpg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
	case "png":
		encoder := &png.Encoder{CompressionLevel: png.BestCompression}
		return encoder.Encode(file, img)
	case "webp":
		// WEBP encoding requires additional library, convert to JPEG for now
		// For full WEBP support, use github.com/chai2010/webp
		return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
	case "bmp":
		return bmp.Encode(file, img)
	case "tiff", "tif":
		return tiff.Encode(file, img, nil)
	default:
		// Default to JPEG
		return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
	}
}

// SaveAsSVG saves image as SVG format (embedded via base64)
func SaveAsSVG(img image.Image, path string) error {
	// Encode image as PNG base64
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create SVG content
	svgContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d">
  <image x="0" y="0" width="%d" height="%d" xlink:href="data:image/png;base64,%s"/>
</svg>`, width, height, width, height, base64Str)

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	return os.WriteFile(path, []byte(svgContent), 0644)
}
