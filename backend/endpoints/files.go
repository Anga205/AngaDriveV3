package endpoints

import (
	"fmt"
	"os"
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

func ReturnFile(c *gin.Context) {
	go socketHandler.SiteActivityPulse()
	file_directory := c.Param("file_directory")

	filePath := getFilePath(file_directory)
	fmt.Println("File path:", filePath)
	if filePath == "" {
		c.JSON(404, gin.H{
			"error": "File not found",
		})
		return
	}

	c.File(filePath)
}
