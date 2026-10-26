package endpoints

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/storage"

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
	filePath := getFilePath(file_directory)
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
