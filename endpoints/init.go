package endpoints

import (
	"angadrive/globals"
	"angadrive/previews"

	"github.com/gin-gonic/gin"
)

func InitEndpoints(r *gin.Engine) {
	setupUploaderRoutes(r)
	r.GET("/i/:file_directory", func(c *gin.Context) {
		if c.Request.Host == globals.AssetsURL {
			returnFile(c)
		}
	})
	r.GET("/i/:file_directory/:original_name", func(c *gin.Context) {
		if c.Request.Host == globals.AssetsURL {
			returnNamedFile(c)
		}
	})
	r.GET("/preview/:file_id", func(c *gin.Context) {
		if c.Request.Host == globals.AssetsURL {
			previews.ReturnPDFPreview(c)
		}
	})
	r.GET("/preview-image/:file_id", func(c *gin.Context) {
		if c.Request.Host == globals.AssetsURL {
			previews.ReturnImagePreview(c)
		}
	})
	r.GET("/preview-video/:file_directory", func(c *gin.Context) {
		if c.Request.Host == globals.AssetsURL {
			previews.ReturnVideoPreview(c)
		}
	})
	r.GET("/download/:file_directory", func(c *gin.Context) {
		if c.Request.Host == globals.AssetsURL {
			downloadFile(c)
		}
	})
}
