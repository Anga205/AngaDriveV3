package endpoints

import (
	"angadrive/config"
	"angadrive/previews"

	"github.com/gin-gonic/gin"
)

// assetsOnly wraps a handler so it only runs when the request targets the
// assets host; otherwise it returns 404.
func assetsOnly(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Host != config.AssetsURL {
			c.AbortWithStatus(404)
			return
		}
		h(c)
	}
}

func InitEndpoints(r *gin.Engine) {
	setupUploaderRoutes(r)
	r.GET("/i/:file_directory", assetsOnly(returnFile))
	r.GET("/i/:file_directory/:original_name", assetsOnly(returnNamedFile))
	r.GET("/preview/:file_id", assetsOnly(previews.ReturnPDFPreview))
	r.GET("/preview-image/:file_id", assetsOnly(previews.ReturnImagePreview))
	r.GET("/preview-video/:file_directory", assetsOnly(previews.ReturnVideoPreview))
	r.GET("/download/:file_directory", assetsOnly(downloadFile))
}
