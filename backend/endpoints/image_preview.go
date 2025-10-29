package endpoints

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"service/database"
	"service/socketHandler"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/jdeng/goheif"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func returnImagePreview(c *gin.Context) {
	go socketHandler.SiteActivityPulse()

	fileDirectory := c.Param("file_directory")

	// this creates: /uploaded_files/image_previews
	previewsDir := filepath.Join(UPLOAD_DIR, "image_previews")
	// this creates: /uploaded_files/image_previews/<file_directory>
	previewFile := filepath.Join(previewsDir, fileDirectory)

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

	originalFilePath := filepath.Join(UPLOAD_DIR, "i", fileInfo.Md5sum)
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

	imageHeight := img.Bounds().Dy()
	imageWidth := img.Bounds().Dx()

	var newHeight, newWidth int
	if imageHeight > imageWidth {
		ratioOfConversion := float64(imageHeight) / 512.0
		newWidth = int(float64(imageWidth) / ratioOfConversion)
		newHeight = 512
	} else {
		ratioOfConversion := float64(imageWidth) / 512.0
		newHeight = int(float64(imageHeight) / ratioOfConversion)
		newWidth = 512
	}

	resizedImg := imaging.Thumbnail(img, newWidth, newHeight, imaging.Lanczos)

	var buf bytes.Buffer

	switch ext {
	case ".jpg", ".jpeg":
		if err := jpeg.Encode(&buf, resizedImg, nil); err != nil {
			return fmt.Errorf("failed to encode jpeg: %w", err)
		}
	case ".png", ".heic", ".heif": // HEIC will be encoded as PNG preview
		if err := png.Encode(&buf, resizedImg); err != nil {
			return fmt.Errorf("failed to encode png: %w", err)
		}
	case ".gif":
		if err := gif.Encode(&buf, resizedImg, nil); err != nil {
			return fmt.Errorf("failed to encode gif: %w", err)
		}
	case ".bmp":
		if err := bmp.Encode(&buf, resizedImg); err != nil {
			return fmt.Errorf("failed to encode bmp: %w", err)
		}
	case ".tiff":
		if err := tiff.Encode(&buf, resizedImg, nil); err != nil {
			return fmt.Errorf("failed to encode tiff: %w", err)
		}
	case ".webp":
		// Note: Standard library does not support encoding webp.
		// Using a third-party library would be needed for full webp support.
		// For now, we can encode it as PNG as a fallback.
		if err := png.Encode(&buf, resizedImg); err != nil {
			return fmt.Errorf("failed to encode webp as png: %w", err)
		}
	default:
		return fmt.Errorf("unsupported image format: %s", fileInfo.OriginalFileName)
	}

	if int64(buf.Len()) > fileInfo.FileSize {
		// If the generated preview is larger than the original, copy the original file instead
		originalFile, err := os.Open(originalFilePath)
		if err != nil {
			return fmt.Errorf("failed to open original file for copying: %w", err)
		}
		defer originalFile.Close()

		outFile, err := os.Create(previewFilePath)
		if err != nil {
			return fmt.Errorf("failed to create preview file for copying: %w", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, originalFile)
		if err != nil {
			return fmt.Errorf("failed to copy original file to preview path: %w", err)
		}
	} else {
		// Otherwise, write the generated preview
		outFile, err := os.Create(previewFilePath)
		if err != nil {
			return fmt.Errorf("failed to create preview file: %w", err)
		}
		defer outFile.Close()

		_, err = outFile.Write(buf.Bytes())
		if err != nil {
			return fmt.Errorf("failed to write preview to file: %w", err)
		}
	}

	return nil
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
