package endpoints

import "github.com/gin-gonic/gin"

func InitEndpoints(r *gin.Engine, UPLOAD_DIR string) {
	setupUploaderRoutes(r, UPLOAD_DIR)
	r.POST("/user/files", getUserFilesHTTP)
	r.GET("/i/:file_directory", returnFile)
	r.GET("/i/:file_directory/:original_name", returnNamedFile)
}
