package endpoints

import (
	"os"
	"path/filepath"
	"service/database"
	"service/socketHandler"

	"github.com/gin-gonic/gin"
)

func getFilePath(file_directory string) string {
	File, err := database.GetFile(file_directory)
	if err != nil {
		return ""
	}
	return UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + File.Md5sum
}

func returnFile(c *gin.Context) {
	go socketHandler.SiteActivityPulse()
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
	go socketHandler.SiteActivityPulse()
	file_directory := c.Param("file_directory")
	original_name := c.Param("original_name")
	filepath := getFilePath(file_directory + filepath.Ext(original_name))
	if filepath == "" {
		c.JSON(404, gin.H{
			"error": "File not found",
		})
		return
	}
	c.File(filepath)
}
