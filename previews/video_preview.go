package previews

import (
	"angadrive/database"
	"angadrive/globals"
	"angadrive/requestHandler"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ReturnVideoPreview serves a GIF preview of a video file.
//
// The flow is:
//  1. Look up the file in the database to get its sha256sum.
//  2. Check the video_previews directory for a preview named <sha256>.gif.
//  3. If it exists, serve it.
//  4. If it does not exist, return HTTP 425 (Too Early) indicating the preview
//     is being generated, and enqueue a runner to generate it.
//
// The runner is deduplicated by the source file's sha256, so repeated requests
// for the same video while a preview is pending will not enqueue duplicate
// jobs.
func ReturnVideoPreview(c *gin.Context) {
	go requestHandler.SiteActivityPulse()

	fileDirectory := c.Param("file_directory")
	// The route is /preview-video/<file_directory>.gif; strip the extension.
	fileDirectory = strings.TrimSuffix(fileDirectory, ".gif")

	file, err := database.GetFile(fileDirectory)
	if err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	previewsDir := filepath.Join(globals.UPLOAD_DIR, "video_previews")
	previewFile := filepath.Join(previewsDir, file.Sha256sum+".gif")

	if _, err := os.Stat(previewFile); !os.IsNotExist(err) {
		c.File(previewFile)
		return
	}

	// Preview does not exist yet. Enqueue a runner to generate it. If a job is
	// already queued/running for this source video, Submit returns an error
	// (dedup by sha256), which is fine — the preview is already being made.
	_ = requestHandler.SubmitVideoPreview(file)

	c.Status(http.StatusTooEarly) // 425
	c.String(http.StatusTooEarly, "Preview is being generated")
}
