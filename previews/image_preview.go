package previews

import (
	"angadrive/database"
	"angadrive/globals"
	"angadrive/requestHandler"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/jdeng/goheif"
	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// previewMaxDimension is the longest edge (in pixels) of generated previews.
const previewMaxDimension = 512

// previewJPEGQuality is the JPEG quality used for opaque previews. Minor
// quality loss is acceptable since these are thumbnails, and a lower quality
// keeps the preview small.
const previewJPEGQuality = 80

func ReturnImagePreview(c *gin.Context) {
	go requestHandler.SiteActivityPulse()

	fileDirectory := c.Param("file_id")

	file, err := database.GetFile(fileDirectory)
	if err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	// SVG files are served raw (no rasterization needed), with a size limit.
	if strings.ToLower(filepath.Ext(fileDirectory)) == ".svg" {
		serveRawSVG(c, file.Sha256sum)
		return
	}

	// this creates: /uploaded_files/image_previews
	previewsDir := filepath.Join(globals.UPLOAD_DIR, "image_previews")
	// this creates: /uploaded_files/image_previews/<file_directory>
	previewFile := filepath.Join(previewsDir, file.Sha256sum)

	if _, err := os.Stat(previewFile); !os.IsNotExist(err) {
		c.File(previewFile)
		return
	}

	if err := generateImagePreview(fileDirectory, previewsDir, previewFile); err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate preview: "+err.Error())
		return
	}
	c.File(previewFile)
}

func generateImagePreview(fileDirectory string, previewsDir string, previewFilePath string) error {
	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create previews directory: %w", err)
	}

	fileInfo, err := database.GetFile(fileDirectory)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	originalFilePath := filepath.Join(globals.UPLOAD_DIR, "i", fileInfo.Sha256sum)
	file, err := os.Open(originalFilePath)
	if err != nil {
		return fmt.Errorf("failed to open original file: %w", err)
	}
	defer file.Close()

	var img image.Image

	ext := strings.ToLower(filepath.Ext(originalFilePath))
	switch ext {
	case ".heic", ".heif":
		img, err = decodeHEIC(file)
		if err != nil {
			return fmt.Errorf("failed to decode HEIC/HEIF: %w", err)
		}
	default:
		img, err = correctImageOrientation(file)
		if err != nil {
			return fmt.Errorf("failed to correct image orientation: %w", err)
		}
	}

	// Always produce an actual thumbnail: resize so the longest dimension is at
	// most previewMaxDimension, preserving aspect ratio.
	resizedImg := resizeToMaxDimension(img)

	// Write to a temporary file so a partially-written preview is never exposed
	// to another request.
	tempFile, err := os.CreateTemp(previewsDir, ".preview-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary preview: %w", err)
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if imageHasTransparency(resizedImg) {
		// Transparent/translucent pixels require a lossless format to preserve
		// alpha. PNG with maximum compression keeps the preview small.
		if err := imaging.Encode(tempFile, resizedImg, imaging.PNG, imaging.PNGCompressionLevel(png.BestCompression)); err != nil {
			tempFile.Close()
			return fmt.Errorf("failed to encode PNG preview: %w", err)
		}
	} else {
		// Fully opaque images are encoded as JPEG. Minor quality loss is
		// acceptable for previews, and the lower quality keeps the file small.
		if err := imaging.Encode(tempFile, resizedImg, imaging.JPEG, imaging.JPEGQuality(previewJPEGQuality)); err != nil {
			tempFile.Close()
			return fmt.Errorf("failed to encode JPEG preview: %w", err)
		}
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync preview: %w", err)
	}

	// Compare the generated preview against the original file.
	previewInfo, err := tempFile.Stat()
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to stat generated preview: %w", err)
	}

	originalInfo, err := os.Stat(originalFilePath)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to stat original file: %w", err)
	}

	// If the original is smaller, use the original instead.
	if originalInfo.Size() <= previewInfo.Size() {
		if err := tempFile.Close(); err != nil {
			return fmt.Errorf("failed to close temporary preview: %w", err)
		}

		os.Remove(tempPath)

		if err := copyFile(originalFilePath, previewFilePath); err != nil {
			return fmt.Errorf("failed to copy original file as preview: %w", err)
		}

		return nil
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close preview: %w", err)
	}

	if err := os.Rename(tempPath, previewFilePath); err != nil {
		return fmt.Errorf("failed to finalize preview: %w", err)
	}
	return nil
}

// resizeToMaxDimension resizes img so its longest dimension is at most
// previewMaxDimension pixels, preserving aspect ratio. Images already within
// the limit are returned unchanged (converted to NRGBA for consistent
// handling).
func resizeToMaxDimension(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= previewMaxDimension && height <= previewMaxDimension {
		return imaging.Thumbnail(img, width, height, imaging.Lanczos)
	}

	if height > width {
		ratio := float64(height) / float64(previewMaxDimension)
		newWidth := int(float64(width) / ratio)
		if newWidth < 1 {
			newWidth = 1
		}
		return imaging.Thumbnail(img, newWidth, previewMaxDimension, imaging.Lanczos)
	}

	ratio := float64(width) / float64(previewMaxDimension)
	newHeight := int(float64(height) / ratio)
	if newHeight < 1 {
		newHeight = 1
	}
	return imaging.Thumbnail(img, previewMaxDimension, newHeight, imaging.Lanczos)
}

// imageHasTransparency reports whether img contains any pixel that is not fully
// opaque (alpha < 0xffff). Such images must be encoded as PNG to preserve
// their alpha channel.
func imageHasTransparency(img image.Image) bool {
	if nrgba, ok := img.(*image.NRGBA); ok {
		return !nrgba.Opaque()
	}
	// Generic fallback for other image types.
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return true
			}
		}
	}
	return false
}

func serveRawSVG(c *gin.Context, fileDirectory string) {
	fileInfo, err := database.GetFile(fileDirectory)
	if err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	if fileInfo.FileSize > 250*1024 {
		c.String(http.StatusBadRequest, "SVG file exceeds 250KB preview limit")
		return
	}

	originalFilePath := filepath.Join(globals.UPLOAD_DIR, "i", fileInfo.Sha256sum)
	c.File(originalFilePath)
}

func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	return err
}

func decodeHEIC(file *os.File) (image.Image, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	// Decode HEIC image
	img, err := goheif.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIC: %w", err)
	}

	// Extract EXIF metadata (contains orientation)
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	exifData, err := goheif.ExtractExif(file)
	if err != nil || exifData == nil {
		// No EXIF, return as-is
		return img, nil
	}

	x, err := exif.Decode(bytes.NewReader(exifData))
	if err != nil || x == nil {
		return img, nil
	}

	// Try reading orientation
	orientTag, err := x.Get(exif.Orientation)
	if err != nil {
		return img, nil
	}

	orientation, err := orientTag.Int(0)
	if err != nil {
		return img, nil
	}

	// Apply same orientation corrections
	switch orientation {
	case 2:
		img = imaging.FlipH(img)
	case 3:
		img = imaging.Rotate180(img)
	case 4:
		img = imaging.FlipV(img)
	case 5:
		img = imaging.Transpose(img)
	case 6:
		img = imaging.Rotate270(img)
	case 7:
		img = imaging.Transverse(img)
	case 8:
		img = imaging.Rotate90(img)
	}

	return img, nil
}

func correctImageOrientation(file *os.File) (image.Image, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	// Try EXIF decode (JPEG/TIFF/etc.)
	x, exifErr := exif.Decode(file)

	// Reset for image decode
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	// If no EXIF, just return the image
	if exifErr != nil || x == nil {
		return img, nil
	}

	// Try to read orientation
	orient, err := x.Get(exif.Orientation)
	if err != nil {
		return img, nil
	}
	orientation, err := orient.Int(0)
	if err != nil {
		return img, nil
	}

	// Apply orientation corrections
	switch orientation {
	case 2:
		img = imaging.FlipH(img)
	case 3:
		img = imaging.Rotate180(img)
	case 4:
		img = imaging.FlipV(img)
	case 5:
		img = imaging.Transpose(img)
	case 6:
		img = imaging.Rotate270(img)
	case 7:
		img = imaging.Transverse(img)
	case 8:
		img = imaging.Rotate90(img)
	}

	return img, nil
}
