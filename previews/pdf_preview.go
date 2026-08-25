package previews

import (
	"angadrive/database"
	"angadrive/globals"
	"angadrive/requestHandler"
	"fmt"
	"image/jpeg"
	"os"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

func ReturnPDFPreview(c *gin.Context) {
	go requestHandler.SiteActivityPulse()

	file_directory := c.Param("file_id")
	file_directory = strings.TrimSuffix(file_directory, ".jpg")
	file, err := database.GetFile(file_directory)
	if err != nil {
		c.String(404, "File not found")
		return
	}
	previewsDir := globals.UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews"
	previewFile := previewsDir + string(os.PathSeparator) + file.Sha256sum + ".jpg"

	if _, err := os.Stat(previewFile); !os.IsNotExist(err) {
		c.File(previewFile)
	} else {
		err := generatePDFPreview(file, previewsDir, previewFile)
		if err != nil {
			c.String(500, "Failed to generate preview: "+err.Error())
			return
		}
		c.File(previewFile)
	}
}

func generatePDFPreview(file database.FileData, previewsDir string, previewFilePath string) error {
	doc, err := fitz.New(globals.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + file.Sha256sum)
	if err != nil {
		return fmt.Errorf("failed to open PDF document: %w", err)
	}
	defer doc.Close()

	if doc.NumPage() < 1 {
		return fmt.Errorf("PDF document has no pages")
	}

	img, err := doc.Image(0)
	if err != nil {
		return fmt.Errorf("failed to extract image from PDF: %w", err)
	}

	// Resize so the longest dimension is at most 512px.
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var newWidth, newHeight uint

	if width >= height {
		newWidth = 512
		newHeight = uint(float64(height) * 512.0 / float64(width))
	} else {
		newHeight = 512
		newWidth = uint(float64(width) * 512.0 / float64(height))
	}

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	resized := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)

	// PDF pages are rendered as opaque images, so JPEG is appropriate.
	// Quality 82 provides a good preview/size tradeoff.
	jpegImage := resized

	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create previews directory: %w", err)
	}

	// Write to a temporary file so a partially-written preview is never
	// exposed to another request.
	tempFile, err := os.CreateTemp(previewsDir, ".preview-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary preview: %w", err)
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := jpeg.Encode(tempFile, jpegImage, &jpeg.Options{
		Quality: 75,
	}); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to encode JPEG preview: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync JPEG preview: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close JPEG preview: %w", err)
	}

	if err := os.Rename(tempPath, previewFilePath); err != nil {
		return fmt.Errorf("failed to finalize JPEG preview: %w", err)
	}

	return nil
}
