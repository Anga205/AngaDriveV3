package endpoints

import (
	"net/http"
	"service/socketHandler"

	"github.com/gin-gonic/gin"
)

func getUserFilesHTTP(c *gin.Context) {
	var req socketHandler.AuthRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	files, err := socketHandler.GetUserFiles(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, files)
}
