package previews

import (
	"angadrive/database"
	"angadrive/socketHandler"
	"angadrive/vars"
	"bytes"
	"fmt"
	"image/png"
	"os"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

func ReturnPDFPreview(c *gin.Context) {
	go socketHandler.SiteActivityPulse()

	file_directory := c.Param("file_id")
	file_directory = strings.TrimSuffix(file_directory, ".png")
	file, err := database.GetFile(file_directory)
	if err != nil {
		c.String(404, "File not found")
		return
	}
	previewsDir := vars.UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews"
	previewFile := previewsDir + string(os.PathSeparator) + file.Sha256sum + ".png"

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
	doc, err := fitz.New(vars.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + file.Sha256sum)
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

	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create previews directory: %w", err)
	}

	f, err := os.Create(previewFilePath)
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
