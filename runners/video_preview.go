package runners

import (
	"angadrive/database"
	"angadrive/globals"
	"bytes"
	"fmt"
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
	// rates reduce file size (fewer frames to encode); 24fps is a good balance
	// between smoothness and size for a thumbnail preview.
	videoPreviewFPS = 24
	// videoPreviewMaxDim is the maximum width/height (pixels) of a generated
	// WebP preview. The aspect ratio of the source is preserved.
	videoPreviewMaxDim = 350
	// videoPreviewQuality is the libwebp_anim quality (0-100). WebP is not
	// limited to a 256-colour palette, so a high quality still produces a much
	// smaller file than the equivalent GIF.
	videoPreviewQuality = 80
	// videoPreviewCompressionLevel is the libwebp_anim encoder effort (0-6).
	// Higher values spend more CPU for a smaller output.
	videoPreviewCompressionLevel = 6
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
// fail. In that case an empty WebP is written to the output path so the file is
// marked as "cannot be previewed" and the runner will not attempt to generate
// it again (the preview endpoint sees the file exists and serves it).
func (r *VideoPreviewRunner) generatePreview(inputFile database.FileData) error {
	inputFilePath := filepath.Join(globals.UPLOAD_DIR, "i", inputFile.Sha256sum)
	previewsDir := filepath.Join(globals.UPLOAD_DIR, "video_previews")
	outputFilePath := filepath.Join(previewsDir, inputFile.Sha256sum+".webp")

	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "input file not found: " + inputFile.OriginalFileName,
			},
		})
		return fmt.Errorf("input file not found: %s", inputFile.OriginalFileName)
	}

	duration, err := getVideoDuration(inputFilePath)
	if err != nil {
		// Corrupted or non-video file: mark it with an empty WebP so we never
		// try again, and notify the owner.
		r.writeEmptyWebP(outputFilePath)
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to read video duration: " + err.Error(),
			},
		})
		return err
	}

	// If the source is longer than the max preview length, speed it up so the
	// preview is exactly the max length (never more).
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
		// Generation failed (corrupted/unsupported source). Mark it with an
		// empty WebP so we never try again, and notify the owner.
		r.writeEmptyWebP(outputFilePath)
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to generate video preview: " + err.Error(),
			},
		})
		return err
	}

	if err := os.Rename(tempPath, outputFilePath); err != nil {
		return fmt.Errorf("failed to finalize video preview: %w", err)
	}
	return nil
}

// writeEmptyWebP writes a minimal valid 1x1 transparent WebP to the given path.
// It is used to mark a video whose preview could not be generated (e.g. a
// corrupted or non-video file), so the backfill loop and preview endpoint stop
// trying to generate a preview for it.
func (r *VideoPreviewRunner) writeEmptyWebP(path string) {
	// A minimal 1x1 transparent lossless WebP (34 bytes).
	emptyWebP := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x1a, 0x00, 0x00, 0x00, // file size - 8
		0x57, 0x45, 0x42, 0x50, // "WEBP"
		0x56, 0x50, 0x38, 0x4c, // "VP8L"
		0x0d, 0x00, 0x00, 0x00, // chunk size
		0x2f, 0x00, 0x00, 0x00, // signature
		0x10, 0x07, 0x10, 0x11, 0x11, 0x88, 0x88, 0xfe, 0x07, // image data
		0x00, // padding
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return
	}
	_ = os.WriteFile(path, emptyWebP, 0o644)
}

// getVideoDuration returns the duration (seconds) of a video using ffprobe.
func getVideoDuration(inputPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err != nil {
		return 0, err
	}
	return duration, nil
}

// generateWebP produces an animated WebP from a video using ffmpeg's
// libwebp_anim encoder. Unlike GIF, WebP is not limited to a 256-colour
// palette, so colours are far more accurate and the file compresses much
// better.
//
// The filter scales to fit within 350x350 (preserving aspect ratio), runs at
// 24fps, and speeds up the video by the given factor so the output is at most
// 15 seconds.
//
// File size is minimized while preserving quality by:
//   - Using -quality 80, which keeps high colour fidelity at a fraction of
//     GIF's size.
//   - Using -compression_level 6, the maximum encoder effort.
//   - Looping forever (-loop 0), like a thumbnail should.
func generateWebP(inputPath, outputPath string, speed float64) error {
	filter := fmt.Sprintf(
		"setpts=PTS/%v,fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease:flags=lanczos",
		speed, videoPreviewFPS, videoPreviewMaxDim, videoPreviewMaxDim,
	)
	maxDuration := strconv.FormatFloat(videoPreviewMaxDuration, 'f', -1, 64)

	cmd := exec.Command("ffmpeg", "-y", "-v", "error",
		"-i", inputPath,
		"-vf", filter,
		"-t", maxDuration,
		"-c:v", "libwebp_anim",
		"-quality", strconv.Itoa(videoPreviewQuality),
		"-compression_level", strconv.Itoa(videoPreviewCompressionLevel),
		"-loop", "0", // always loop for a thumbnail preview
		"-an", // discard any audio stream
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("webp generation failed: %v: %s", err, stderr.String())
	}
	return nil
}
