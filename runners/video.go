package runners

import (
	"angadrive/database"
	"angadrive/globals"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// VideoJob is a request to convert a video file to MP4.
type VideoJob struct {
	File database.FileData
}

// ID returns the dedup key for the job (the file's content hash).
func (j VideoJob) ID() string {
	return j.File.Sha256sum
}

// VideoRunner converts video files to MP4 using ffmpeg.
//
// It is the first concrete Runner built on the generic runners framework. The
// actual transcoding is CPU-intensive, so it is queued and run with bounded
// concurrency by the Manager.
type VideoRunner struct {
	notifier Notifier
}

// Name returns the unique identifier used to submit video jobs.
func (r *VideoRunner) Name() string {
	return "video"
}

// Run converts a single video file. It is called from a worker goroutine.
func (r *VideoRunner) Run(job Job) error {
	vj, ok := job.(VideoJob)
	if !ok {
		return fmt.Errorf("video runner received unexpected job type %T", job)
	}
	return r.convert(vj.File)
}

func removeExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[:i]
		}
	}
	return filename
}

func sha256sum(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// convert performs the actual ffmpeg transcode and notifies the user of the
// result (or error) through the Notifier.
func (r *VideoRunner) convert(inputFile database.FileData) error {
	inputFilePath := globals.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + inputFile.Sha256sum
	outputFilePath := globals.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + removeExtension(inputFile.Sha256sum) + ".mp4"
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "input file not found: " + inputFile.OriginalFileName,
			},
		})
		return fmt.Errorf("input file not found: %s", inputFile.OriginalFileName)
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputFilePath,
		"-c:v", "libx264",
		"-profile:v", "baseline",
		"-level", "3.1",
		"-pix_fmt", "yuv420p",
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		"-preset", "medium",
		"-crf", "20",
		"-movflags", "+faststart",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-ac", "2",
		outputFilePath,
	)

	// Each conversion needs its own stderr buffer to avoid race conditions.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		go os.Remove(outputFilePath)
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": fmt.Sprintf("failed to convert video: %v. FFMpeg output: %s", err, stderr.String()),
			},
		})
		return err
	}
	fileInfo, err := os.Stat(outputFilePath)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to get output file stats: " + err.Error(),
			},
		})
		return err
	}
	fileSize := fileInfo.Size()
	uniqueFileName := database.GenerateUniqueFileName(inputFile.OriginalFileName + ".mp4")
	outputSha256sum, err := sha256sum(outputFilePath)
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to calculate SHA-256 checksum: " + err.Error(),
			},
		})
		return err
	}
	err = os.Rename(outputFilePath, globals.UPLOAD_DIR+string(os.PathSeparator)+"i"+string(os.PathSeparator)+outputSha256sum+".mp4")
	if err != nil {
		r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to rename output file: " + err.Error(),
			},
		})
		return err
	}
	fileData := database.FileData{
		OriginalFileName: removeExtension(inputFile.OriginalFileName) + ".mp4",
		FileDirectory:    uniqueFileName,
		AccountToken:     inputFile.AccountToken,
		FileSize:         fileSize,
		Timestamp:        time.Now().UTC().Unix(),
		Sha256sum:        outputSha256sum + ".mp4",
	}

	fileData.Insert()
	r.notifier.NotifyFileAdded(fileData)
	r.notifier.NotifyUser(inputFile.AccountToken, map[string]interface{}{
		"type": "convert_video_response",
		"data": map[string]interface{}{
			"file": fileData,
		},
	})
	return nil
}
