package runners

import (
	"angadrive/database"
	"angadrive/globals"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// videoExtensions are the file extensions treated as videos for preview
// generation.
var videoExtensions = []string{"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"}

// isVideoFile reports whether the given file name has a video extension.
func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, v := range videoExtensions {
		if ext == "."+v {
			return true
		}
	}
	return false
}

// previewExists reports whether a preview (or an empty "cannot preview" marker)
// already exists for the given source sha256.
func previewExists(sha256sum string) bool {
	path := filepath.Join(globals.UPLOAD_DIR, "video_previews", sha256sum+".gif")
	_, err := os.Stat(path)
	return err == nil
}

// StartBackfillLoop begins a background loop that keeps the CPU busy generating
// video previews until every video in the database has a preview.
//
// The loop only starts a new job when the manager is completely idle (no jobs
// queued and none running). It scans the database for the first video that does
// not yet have a preview, submits exactly one preview job, and waits for the
// manager to become idle again before scanning again. This ensures the CPU is
// only idle when there are no pending previews left.
func StartBackfillLoop(m *Manager) {
	go func() {
		for {
			// Wait until the manager is completely idle.
			m.WaitIdle()

			// Scan the database for a video without a preview.
			job, found := findNextVideoWithoutPreview()
			if !found {
				// All videos have previews; nothing to do. Sleep briefly and
				// re-check in case new videos are uploaded.
				time.Sleep(5 * time.Second)
				continue
			}

			// Submit exactly one job, then loop back to wait for idle.
			if err := m.Submit("video_preview", job); err != nil {
				// If it's already queued/running, just wait for idle again.
				time.Sleep(time.Second)
			}
		}
	}()
}

// findNextVideoWithoutPreview scans the database and returns the first video
// file that does not yet have a preview. It returns found=false if every video
// already has a preview.
func findNextVideoWithoutPreview() (VideoPreviewJob, bool) {
	files, err := database.GetAllFiles()
	if err != nil {
		return VideoPreviewJob{}, false
	}
	for _, file := range files {
		if !isVideoFile(file.OriginalFileName) {
			continue
		}
		if previewExists(file.Sha256sum) {
			continue
		}
		return VideoPreviewJob{File: file}, true
	}
	return VideoPreviewJob{}, false
}

// WaitIdle blocks until the manager has no queued jobs and no running jobs.
func (m *Manager) WaitIdle() {
	for {
		if m.IsIdle() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// IsIdle reports whether the manager has no queued jobs and no running jobs.
func (m *Manager) IsIdle() bool {
	return len(m.queue) == 0 && m.activeJobs() == 0
}

// activeJobs returns the number of jobs currently running (holding a semaphore
// slot).
func (m *Manager) activeJobs() int {
	return len(m.semaphore)
}
