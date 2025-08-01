package endpoints

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"service/database"
	"service/socketHandler"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

func returnFilePreview(c *gin.Context) {
	go socketHandler.SiteActivityPulse()

	file_directory := c.Param("file_directory")
	previewsDir := UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews"
	previewFile := previewsDir + string(os.PathSeparator) + file_directory

	if _, err := os.Stat(previewFile); !os.IsNotExist(err) {
		c.File(previewFile)
	} else {
		err := generatePreview(file_directory)
		if err != nil {
			c.String(500, "Failed to generate preview: "+err.Error())
			return
		}
		c.File(previewFile)
	}
}

func generatePreview(file_directory string) error {
	previewFile := UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews" + string(os.PathSeparator) + file_directory
	if !strings.HasSuffix(file_directory, ".png") {
		return fmt.Errorf("file must be a PNG image")
	}
	file_directory_without_png := strings.TrimSuffix(file_directory, ".png")
	if !strings.HasSuffix(file_directory_without_png, ".pdf") {
		return fmt.Errorf("file must be a PDF document")
	}

	file_info, err := database.GetFile(file_directory_without_png)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	doc, err := fitz.New(UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + file_info.Md5sum)
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

	// Calculate new dimensions while preserving aspect ratio
	originalBounds := img.Bounds()
	originalWidth := originalBounds.Dx()
	originalHeight := originalBounds.Dy()

	var newWidth, newHeight uint
	if originalWidth > originalHeight {
		newWidth = 512
		newHeight = uint(float64(originalHeight) * (512.0 / float64(originalWidth)))
	} else {
		newHeight = 512
		newWidth = uint(float64(originalWidth) * (512.0 / float64(originalHeight)))
	}

	// If for some reason a dimension is 0, make it 1 to avoid errors
	if newWidth == 0 {
		newWidth = 1
	}
	if newHeight == 0 {
		newHeight = 1
	}

	resized := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	err = encoder.Encode(&buf, resized)
	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	previewsDir := UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews"
	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create previews directory: %w", err)
	}

	f, err := os.Create(previewFile)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write image to file: %w", err)
	}
	return nil
}
