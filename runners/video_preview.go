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
	// videoPreviewFPS is the frame rate of generated GIF previews.
	videoPreviewFPS = 24
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

// generateGIF produces a GIF from a video using a two-pass palette approach for
// good quality and broad compatibility. The filter scales to fit within
// 512x512 (preserving aspect ratio), runs at 24fps, and speeds up the video by
// the given factor so the output is at most 15 seconds.
func generateGIF(inputPath, outputPath string, speed float64) error {
	filter := fmt.Sprintf(
		"setpts=PTS/%v,fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease:flags=lanczos",
		speed, videoPreviewFPS, videoPreviewMaxDim, videoPreviewMaxDim,
	)
	maxDuration := strconv.FormatFloat(videoPreviewMaxDuration, 'f', -1, 64)

	// Pass 1: generate a palette from the (filtered) frames.
	palettePath := outputPath + ".palette.png"
	defer os.Remove(palettePath)
	paletteCmd := exec.Command("ffmpeg", "-y", "-i", inputPath,
		"-vf", filter+",palettegen",
		"-t", maxDuration,
		palettePath,
	)
	var paletteErr bytes.Buffer
	paletteCmd.Stderr = &paletteErr
	if err := paletteCmd.Run(); err != nil {
		return fmt.Errorf("palette generation failed: %v: %s", err, paletteErr.String())
	}

	// Pass 2: apply the palette to produce the final gif.
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-i", palettePath,
		"-filter_complex", filter+"[x];[x][1:v]paletteuse",
		"-t", maxDuration,
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gif generation failed: %v: %s", err, stderr.String())
	}
	return nil
}
