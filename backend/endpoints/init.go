package endpoints

import (
	"service/vars"

	"github.com/gin-gonic/gin"
)

func InitEndpoints(r *gin.Engine, UPLOAD_DIR string) {
	setupUploaderRoutes(r, UPLOAD_DIR)
	r.GET("/i/:file_directory", func(c *gin.Context) {
		if c.Request.Host == vars.AssetsURL {
			returnFile(c)
		}
	})
	r.GET("/i/:file_directory/:original_name", func(c *gin.Context) {
		if c.Request.Host == vars.AssetsURL {
			returnNamedFile(c)
		}
	})
	r.GET("/preview/:file_directory", func(c *gin.Context) {
		if c.Request.Host == vars.AssetsURL {
			returnFilePreview(c)
		}
	})
}
