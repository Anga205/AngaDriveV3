package previews

import (
	"angadrive/database"
	"angadrive/globals"
	"angadrive/requestHandler"
	"angadrive/runners"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ReturnVideoPreview serves a WebP preview of a video file.
//
// The flow is:
//  1. Look up the file in the database to get its sha256sum.
//  2. Check the video_previews directory for a preview named <sha256>.webp.
//  3. If it exists, serve it.
//  4. If the file is marked as unpreviewable (<sha256>.invalid), respond with
//     HTTP 415 (Unsupported Media Type) so the UI can fall back to a file icon
//     instead of displaying a blank image.
//  5. Otherwise return HTTP 425 (Too Early) indicating the preview is being
//     generated, and enqueue a runner to generate it.
//
// The runner is deduplicated by the source file's sha256, so repeated requests
// for the same video while a preview is pending will not enqueue duplicate
// jobs.
func ReturnVideoPreview(c *gin.Context) {
	go requestHandler.SiteActivityPulse()

	fileDirectory := c.Param("file_directory")
	// The route is /preview-video/<file_directory>.webp; strip the extension.
	fileDirectory = strings.TrimSuffix(fileDirectory, ".webp")

	file, err := database.GetFile(fileDirectory)
	if err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	previewsDir := filepath.Join(globals.UPLOAD_DIR, "video_previews")
	previewFile := filepath.Join(previewsDir, file.Sha256sum+".webp")

	// The frontend fetches previews with fetch() so it can distinguish "still
	// generating" (425) from "cannot be previewed" (415). Assets are served
	// from a different origin than the app, so allow cross-origin reads. The
	// endpoint is already unauthenticated, so this exposes nothing new.
	c.Header("Access-Control-Allow-Origin", "*")

	if info, err := os.Stat(previewFile); err == nil {
		if info.Size() >= runners.MinValidPreviewBytes {
			c.Header("Cache-Control", "private, max-age=900")
			c.File(previewFile)
			return
		}
		// Legacy blank placeholder from an older version: drop it so the file
		// is regenerated (and marked properly if it is still unpreviewable).
		os.Remove(previewFile)
	}

	// The video was already determined to be unpreviewable. Report it as such
	// (and cache the answer) so the UI does not keep retrying.
	if _, err := os.Stat(filepath.Join(previewsDir, file.Sha256sum+runners.InvalidMarkerSuffix)); err == nil {
		c.Header("Cache-Control", "private, max-age=900")
		c.Header("X-Preview-Status", "unavailable")
		c.String(http.StatusUnsupportedMediaType, "Preview unavailable")
		return
	}

	// Preview does not exist yet. Enqueue a runner to generate it. If a job is
	// already queued/running for this source video, Submit returns an error
	// (dedup by sha256), which is fine — the preview is already being made.
	_ = requestHandler.SubmitVideoPreview(file)

	c.Header("Retry-After", "2")
	c.Status(http.StatusTooEarly) // 425
	c.String(http.StatusTooEarly, "Preview is being generated")
}
