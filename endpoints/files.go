package endpoints

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/storage"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func getFilePath(file_directory string) string {
	File, err := database.GetFile(file_directory)
	if err != nil {
		return ""
	}
	return storage.FilePath(File.Sha256sum)
}

func getFileName(file_directory string) string {
	File, err := database.GetFile(file_directory)
	if err != nil {
		return file_directory
	}
	return File.OriginalFileName
}

func returnFile(c *gin.Context) {
	go hub.PublishSiteActivity()
	file_directory := c.Param("file_directory")

	filePath := getFilePath(file_directory)
	if filePath == "" {
		c.JSON(404, gin.H{
			"error": "File not found",
		})
		return
	}

	c.File(filePath)
}

func returnNamedFile(c *gin.Context) {
	go hub.PublishSiteActivity()
	file_directory := c.Param("file_directory")
	original_name := c.Param("original_name")

	// The copy-link URL is built as /i/<file_directory-without-extension>/<original_name>,
	// so the extension must be reconstructed from the original filename. Try the
	// exact directory first (covers callers that keep the extension), then the
	// extension-reconstructed form.
	filePath := getFilePath(file_directory)
	if filePath == "" {
		if ext := filepath.Ext(original_name); ext != "" {
			filePath = getFilePath(file_directory + ext)
		}
	}
	if filePath == "" {
		c.JSON(404, gin.H{
			"error": "File not found",
		})
		return
	}
	c.File(filePath)
}

func downloadFile(c *gin.Context) {
	go hub.PublishSiteActivity()
	file_directory := c.Param("file_directory")

	filePath := getFilePath(file_directory)
	if filePath == "" {
		c.JSON(404, gin.H{
			"error": "File not found",
		})
		return
	}

	c.FileAttachment(filePath, getFileName(file_directory))
}
