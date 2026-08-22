package endpoints

import (
	"angadrive/vars"

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
	r.GET("/preview-image/:file_directory", func(c *gin.Context) {
		if c.Request.Host == vars.AssetsURL {
			returnImagePreview(c)
		}
	})
	r.GET("/download/:file_directory", func(c *gin.Context) {
		// Allow cross-origin requests from WEB_URL
		c.Writer.Header().Set("Access-Control-Allow-Origin", vars.WebURL)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		if c.Request.Host == vars.AssetsURL {
			downloadFile(c)
		}
	})
}
