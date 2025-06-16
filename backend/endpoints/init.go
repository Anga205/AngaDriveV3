package endpoints

import "github.com/gin-gonic/gin"

func InitEndpoints(r *gin.Engine, UPLOAD_DIR string) {
	setupUploaderRoutes(r, UPLOAD_DIR)
	r.POST("/user/files", getUserFilesHTTP)
}
