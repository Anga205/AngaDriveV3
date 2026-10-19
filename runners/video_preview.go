package runners

import (
	"angadrive/database"
	"angadrive/globals"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// videoPreviewMaxDuration is the maximum length (seconds) of a generated
	// WebP preview. Longer source videos are sped up so the preview is exactly
	// this long (never more).
	videoPreviewMaxDuration = 15.0
	// videoPreviewFPS is the frame rate of generated WebP previews. Lower frame
	// rates reduce file size (fewer frames to encode); 24fps
	// is a good balance between smoothness and size for a thumbnail preview.
	videoPreviewFPS = 24
	// videoPreviewMaxDim is the maximum width/height (pixels) of a generated
	// WebP preview. The aspect ratio of the source is preserved.
	videoPreviewMaxDim = 350
	// videoProbeTimeout bounds ffprobe. A malformed file must never hold a
	// worker slot (and therefore the whole backfill loop) indefinitely.
	videoProbeTimeout = 30 * time.Second
	// videoEncodeTimeout bounds ffmpeg for the same reason.
	videoEncodeTimeout = 5 * time.Minute
	// InvalidMarkerSuffix is appended to the source sha256 to record that a
	// video cannot be previewed. It is a sidecar file (not a placeholder image)
	// so the endpoint can report the file as unavailable instead of serving a
	// blank image the UI would happily display.
	InvalidMarkerSuffix = ".invalid"
	// MinValidPreviewBytes is the smallest output accepted as a real preview.
	// ffmpeg can exit 0 while writing an empty file for a container with no
	// decodable frames.
	MinValidPreviewBytes = 100
)

// VideoPreviewJob is a request to generate a WebP preview of a video file.
type VideoPreviewJob struct {
	File database.FileData
}

// ID returns the dedup key for the job (the source file's content hash).
// Because it is keyed on the source sha256, a preview for the same source
// video cannot be queued or generated twice concurrently.
func (j VideoPreviewJob) ID() string {
	return j.File.Sha256sum
}

// VideoPreviewRunner generates WebP previews of video files using ffmpeg.
//
// It is a concrete Runner built on the generic runners framework. Preview
// generation is CPU-intensive, so it is queued and run with bounded
// concurrency by the Manager.
type VideoPreviewRunner struct {
	notifier Notifier
}

// Name returns the unique identifier used to submit video preview jobs.
func (r *VideoPreviewRunner) Name() string {
	return "video_preview"
}

// Run generates a single video preview. It is called from a worker goroutine.
func (r *VideoPreviewRunner) Run(job Job) error {
	vj, ok := job.(VideoPreviewJob)
	if !ok {
		return fmt.Errorf("video preview runner received unexpected job type %T", job)
	}
	return r.generatePreview(vj.File)
}

// generatePreview produces a WebP preview for a video file and writes it to the
// video_previews directory as <sha256>.webp.
//
// If the source video is corrupted or not actually a video, generation will
// fail. In that case an "invalid" marker is written next to the preview path so
// the file is marked as "cannot be previewed": the backfill loop stops picking
// it up and the preview endpoint reports it as unavailable instead of serving a
// blank image.
func (r *VideoPreviewRunner) generatePreview(inputFile database.FileData) error {
	inputFilePath := filepath.Join(globals.UPLOAD_DIR, "i", inputFile.Sha256sum)
	previewsDir := filepath.Join(globals.UPLOAD_DIR, "video_previews")
	outputFilePath := filepath.Join(previewsDir, inputFile.Sha256sum+".webp")

	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		r.notifyError(inputFile, "input file not found: "+inputFile.OriginalFileName)
		return fmt.Errorf("input file not found: %s", inputFile.OriginalFileName)
	}

	duration, err := getVideoDuration(inputFilePath)
	if err != nil {
		return r.markInvalid(inputFile, previewsDir, "failed to read video duration: "+err.Error())
	}
	// A container that parses but reports no usable duration cannot be
	// previewed; treat it as invalid rather than handing it to ffmpeg.
	if math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 {
		return r.markInvalid(inputFile, previewsDir,
			fmt.Sprintf("video has no usable duration (%v)", duration))
	}

	// If the source is longer than the max preview length, speed it up so the
	// webp is exactly the max length (never more).
	speed := 1.0
	if duration > videoPreviewMaxDuration {
		speed = duration / videoPreviewMaxDuration
	}

	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create video previews directory: %w", err)
	}

	// Write to a temp file so a partially-written preview is never served.
	// The temp file keeps a .webp extension so ffmpeg can infer the output
	// format (it cannot infer WebP from an extensionless filename).
	tempPath := filepath.Join(previewsDir, fmt.Sprintf(".preview-%d.tmp.webp", time.Now().UnixNano()))
	defer os.Remove(tempPath)

	if err := generateWebP(inputFilePath, tempPath, speed); err != nil {
		return r.markInvalid(inputFile, previewsDir, "failed to generate video preview: "+err.Error())
	}

	// ffmpeg can exit successfully while producing an empty or truncated file
	// (e.g. a container with no decodable frames). Never cache that as a valid
	// preview.
	info, err := os.Stat(tempPath)
	if err != nil {
		return r.markInvalid(inputFile, previewsDir, "failed to stat generated preview: "+err.Error())
	}
	if info.Size() < MinValidPreviewBytes {
		return r.markInvalid(inputFile, previewsDir,
			fmt.Sprintf("generated preview is too small (%d bytes)", info.Size()))
	}

	if err := os.Rename(tempPath, outputFilePath); err != nil {
		return fmt.Errorf("failed to finalize video preview: %w", err)
	}
	return nil
}

// markInvalid records that a video cannot be previewed and notifies its owner.
// It returns an error so the caller can propagate the failure, but the marker
// itself is what prevents the file from being retried forever.
func (r *VideoPreviewRunner) markInvalid(inputFile database.FileData, previewsDir, reason string) error {
	if err := writeInvalidMarker(previewsDir, inputFile.Sha256sum); err != nil {
		// Without a marker the backfill loop would pick this file up again
		// immediately, so surface the failure loudly.
		r.notifyError(inputFile, "failed to record unpreviewable video: "+err.Error())
		return fmt.Errorf("%s (and failed to write invalid marker: %w)", reason, err)
	}
	r.notifyError(inputFile, reason)
	return errors.New(reason)
}

// notifyError sends an error message to the file's owner.
func (r *VideoPreviewRunner) notifyError(inputFile database.FileData, message string) {
	r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
		"type": "error",
		"data": map[string]interface{}{
			"error": message,
		},
	})
}

// invalidMarkerPath returns the path of the sidecar marker that records a
// video as unpreviewable.
func invalidMarkerPath(previewsDir, sha256sum string) string {
	return filepath.Join(previewsDir, sha256sum+InvalidMarkerSuffix)
}

// writeInvalidMarker atomically writes the sidecar marker for a video whose
// preview could not be generated.
func writeInvalidMarker(previewsDir, sha256sum string) error {
	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return err
	}
	markerPath := invalidMarkerPath(previewsDir, sha256sum)
	tempPath := filepath.Join(previewsDir, fmt.Sprintf(".invalid-%d.tmp", time.Now().UnixNano()))
	if err := os.WriteFile(tempPath, []byte("unpreviewable\n"), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, markerPath); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}

// getVideoDuration returns the duration (seconds) of a video using ffprobe.
func getVideoDuration(inputPath string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), videoProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("ffprobe timed out after %s", videoProbeTimeout)
		}
		return 0, err
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err != nil {
		return 0, err
	}
	return duration, nil
}

// generateWebP produces an animated WebP from a video. WebP stores full 24-bit
// color instead of GIF's 256-color palette, so previews keep gradients and
// detail that palette quantization would lose, while animated WebP is also
// smaller than GIF at the same dimensions and frame rate.
//
// The filter scales to fit within 350x350 (preserving aspect ratio), runs at
// 24fps, and speeds up the video by the given factor so the output is at most
// 15 seconds.
func generateWebP(inputPath, outputPath string, speed float64) error {
	filter := fmt.Sprintf(
		"setpts=PTS/%v,fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease:flags=lanczos",
		speed, videoPreviewFPS, videoPreviewMaxDim, videoPreviewMaxDim,
	)
	maxDuration := strconv.FormatFloat(videoPreviewMaxDuration, 'f', -1, 64)

	ctx, cancel := context.WithTimeout(context.Background(), videoEncodeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-v", "error",
		"-i", inputPath,
		"-vf", filter,
		"-t", maxDuration,
		"-c:v", "libwebp_anim",
		"-lossless", "0",
		"-q:v", "75",
		"-loop", "0", // always loop for a thumbnail preview
		"-an", // discard any audio stream
		"-f", "webp",
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("webp generation timed out after %s", videoEncodeTimeout)
		}
		return fmt.Errorf("webp generation failed: %v: %s", err, stderr.String())
	}
	return nil
}
