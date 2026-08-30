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
	// GIF preview. Longer source videos are sped up so the preview is exactly
	// this long (never more).
	videoPreviewMaxDuration = 15.0
	// videoPreviewFPS is the frame rate of generated GIF previews. Lower frame
	// rates dramatically reduce GIF file size (fewer frames to encode); 12fps
	// is a good balance between smoothness and size for a thumbnail preview.
	videoPreviewFPS = 24
	// videoPreviewMaxColors is the maximum number of colors in the GIF palette.
	// Fewer colors reduce file size; 128 is a good balance for preview quality.
	videoPreviewMaxColors = 128
	// videoPreviewMaxDim is the maximum width/height (pixels) of a generated
	// GIF preview. The aspect ratio of the source is preserved.
	videoPreviewMaxDim = 512
)

// VideoPreviewJob is a request to generate a GIF preview of a video file.
type VideoPreviewJob struct {
	File database.FileData
}

// ID returns the dedup key for the job (the source file's content hash).
// Because it is keyed on the source sha256, a preview for the same source
// video cannot be queued or generated twice concurrently.
func (j VideoPreviewJob) ID() string {
	return j.File.Sha256sum
}

// VideoPreviewRunner generates GIF previews of video files using ffmpeg.
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

// generatePreview produces a GIF preview for a video file and writes it to the
// video_previews directory as <sha256>.gif.
//
// If the source video is corrupted or not actually a video, generation will
// fail. In that case an empty GIF is written to the output path so the file is
// marked as "cannot be previewed" and the runner will not attempt to generate
// it again (the preview endpoint sees the file exists and serves it).
func (r *VideoPreviewRunner) generatePreview(inputFile database.FileData) error {
	inputFilePath := filepath.Join(globals.UPLOAD_DIR, "i", inputFile.Sha256sum)
	previewsDir := filepath.Join(globals.UPLOAD_DIR, "video_previews")
	outputFilePath := filepath.Join(previewsDir, inputFile.Sha256sum+".gif")

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
		// Corrupted or non-video file: mark it with an empty GIF so we never
		// try again, and notify the owner.
		r.writeEmptyGIF(outputFilePath)
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to read video duration: " + err.Error(),
			},
		})
		return err
	}

	// If the source is longer than the max preview length, speed it up so the
	// gif is exactly the max length (never more).
	speed := 1.0
	if duration > videoPreviewMaxDuration {
		speed = duration / videoPreviewMaxDuration
	}

	if err := os.MkdirAll(previewsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create video previews directory: %w", err)
	}

	// Write to a temp file so a partially-written preview is never served.
	// The temp file keeps a .gif extension so ffmpeg can infer the output
	// format (it cannot infer GIF from an extensionless filename).
	tempPath := filepath.Join(previewsDir, fmt.Sprintf(".preview-%d.tmp.gif", time.Now().UnixNano()))
	defer os.Remove(tempPath)

	if err := generateGIF(inputFilePath, tempPath, speed); err != nil {
		// Generation failed (corrupted/unsupported source). Mark it with an
		// empty GIF so we never try again, and notify the owner.
		r.writeEmptyGIF(outputFilePath)
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

// writeEmptyGIF writes a minimal valid 1x1 transparent GIF to the given path.
// It is used to mark a video whose preview could not be generated (e.g. a
// corrupted or non-video file), so the backfill loop and preview endpoint stop
// trying to generate a preview for it.
func (r *VideoPreviewRunner) writeEmptyGIF(path string) {
	// A minimal 1x1 transparent GIF89a.
	emptyGIF := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // "GIF89a"
		0x01, 0x00, 0x01, 0x00, // 1x1
		0x80, 0x00, 0x00, // GCT flag, 1 color
		0x00, 0x00, 0x00, 0x00, // transparent color
		0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // image descriptor
		0x02, 0x02, 0x44, 0x01, 0x00, // image data
		0x3b, // trailer
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return
	}
	_ = os.WriteFile(path, emptyGIF, 0o644)
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

// generateGIF produces a GIF from a video using a single-pass palette
// approach. Both palette generation and mapping are done in one filter graph
// via split, which avoids the timestamp/frame mismatches that can occur when
// running palettegen and paletteuse as two separate ffmpeg passes. The
// resulting GIF is robust across browsers (no "corrupt or truncated" errors).
//
// The filter scales to fit within 512x512 (preserving aspect ratio), runs at
// 24fps, and speeds up the video by the given factor so the output is at most
// 15 seconds.
//
// File size is minimized while preserving quality by:
//   - Using a reduced 128-color palette (palettegen max_colors).
//   - Using stats_mode=diff so the palette is built only from frames where
//     pixels actually change, giving a more accurate color distribution.
//   - Using ordered bayer dithering (bayer_scale=5). Unlike error-diffusion
//     dithering, bayer produces a regular, periodic pattern that GIF's LZW
//     compression encodes much more compactly, with minimal visual loss at
//     preview size.
func generateGIF(inputPath, outputPath string, speed float64) error {
	filter := fmt.Sprintf(
		"setpts=PTS/%v,fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease:flags=lanczos,split[s0][s1];[s0]palettegen=stats_mode=diff:max_colors=%d[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5",
		speed, videoPreviewFPS, videoPreviewMaxDim, videoPreviewMaxDim, videoPreviewMaxColors,
	)
	maxDuration := strconv.FormatFloat(videoPreviewMaxDuration, 'f', -1, 64)

	cmd := exec.Command("ffmpeg", "-y", "-v", "error",
		"-i", inputPath,
		"-filter_complex", filter,
		"-t", maxDuration,
		"-loop", "0", // always loop for a thumbnail preview
		"-gifflags", "+transdiff", // improve compression + compatibility
		"-an", // discard any audio stream
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gif generation failed: %v: %s", err, stderr.String())
	}
	return nil
}
