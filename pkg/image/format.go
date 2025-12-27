package image

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

// Format represents image format
type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
	FormatGIF  Format = "gif"
	FormatWEBP Format = "webp"
	FormatBMP  Format = "bmp"
	FormatTIFF Format = "tiff"
)

// DetectFormat detects image format from reader
func DetectFormat(r io.Reader) Format {
	// Read first bytes to detect format
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	buf = buf[:n]

	if len(buf) < 4 {
		return FormatJPEG // default
	}

	// JPEG: FF D8 FF
	if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return FormatJPEG
	}

	// PNG: 89 50 4E 47
	if len(buf) >= 8 && buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return FormatPNG
	}

	// GIF: 47 49 46 38
	if len(buf) >= 6 && buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46 && buf[3] == 0x38 {
		return FormatGIF
	}

	// WEBP: RIFF...WEBP
	if len(buf) >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP" {
		return FormatWEBP
	}

	// BMP: 42 4D
	if buf[0] == 0x42 && buf[1] == 0x4D {
		return FormatBMP
	}

	// TIFF: 49 49 2A 00 or 4D 4D 00 2A
	if (len(buf) >= 4 && buf[0] == 0x49 && buf[1] == 0x49 && buf[2] == 0x2A && buf[3] == 0x00) ||
		(len(buf) >= 4 && buf[0] == 0x4D && buf[1] == 0x4D && buf[2] == 0x00 && buf[3] == 0x2A) {
		return FormatTIFF
	}

	return FormatJPEG // default
}

// DecodeImage decodes image from reader with specified format
func DecodeImage(r io.Reader, format Format) (image.Image, error) {
	switch format {
	case FormatJPEG, "jpg":
		return jpeg.Decode(r)
	case FormatPNG:
		return png.Decode(r)
	case FormatWEBP:
		return webp.Decode(r)
	case FormatBMP:
		return bmp.Decode(r)
	case FormatGIF:
		img, _, err := image.Decode(r)
		return img, err
	case FormatTIFF, "tif":
		return tiff.Decode(r)
	default:
		// Try auto-detection
		img, _, err := image.Decode(r)
		return img, err
	}
}

// FormatFromExtension gets format from file extension
func FormatFromExtension(ext string) Format {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "jpg", "jpeg":
		return FormatJPEG
	case "png":
		return FormatPNG
	case "gif":
		return FormatGIF
	case "webp":
		return FormatWEBP
	case "bmp":
		return FormatBMP
	case "tiff", "tif":
		return FormatTIFF
	default:
		return FormatJPEG
	}
}

// FormatFromFilename gets format from filename
func FormatFromFilename(filename string) Format {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return FormatJPEG
	}
	return FormatFromExtension(filename[idx:])
}

// IsValidFormat checks if format is valid
func IsValidFormat(format Format) bool {
	switch format {
	case FormatJPEG, FormatPNG, FormatGIF, FormatWEBP, FormatBMP, FormatTIFF:
		return true
	default:
		return false
	}
}
