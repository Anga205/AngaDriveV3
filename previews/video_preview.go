package previews

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/service"
	"angadrive/storage"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// ReturnVideoPreview serves a WebP preview of a video file.
//
// The flow is:
//  1. Look up the file in the database to get its sha256sum.
//  2. Check the video_previews directory for a preview named <sha256>.webp.
//  3. If it exists, serve it.
//  4. If it does not exist, return HTTP 425 (Too Early) indicating the preview
//     is being generated, and enqueue a runner to generate it.
//
// The runner is deduplicated by the source file's sha256, so repeated requests
// for the same video while a preview is pending will not enqueue duplicate
// jobs.
func ReturnVideoPreview(c *gin.Context) {
	go hub.PublishSiteActivity()

	fileDirectory := c.Param("file_directory")
	// The route is /preview-video/<file_directory>.webp; strip the extension.
	fileDirectory = strings.TrimSuffix(fileDirectory, ".webp")

	file, err := database.GetFile(fileDirectory)
	if err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	previewFile := storage.VideoPreviewPath(file.Sha256sum)

	if _, err := os.Stat(previewFile); !os.IsNotExist(err) {
		c.Header("Cache-Control", "private, max-age=900")
		c.File(previewFile)
		return
	}

	// Preview does not exist yet. Enqueue a runner to generate it. If a job is
	// already queued/running for this source video, Submit returns an error
	// (dedup by sha256), which is fine — the preview is already being made.
	_ = service.SubmitVideoPreview(file)

	c.Status(http.StatusTooEarly) // 425
	c.String(http.StatusTooEarly, "Preview is being generated")
}
