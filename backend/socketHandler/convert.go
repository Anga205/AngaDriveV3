package socketHandler

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"service/database"
	"sync"
	"time"
)

func removeExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[:i]
		}
	}
	return filename
}

func md5sum(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

const maxConcurrentConversions = 5
const conversionQueueSize = 100

// conversionJob holds the data needed for a conversion task.
type conversionJob struct {
	inputFile database.FileData
}

var (
	// A map to track files that are currently being converted or are in the queue.
	// This prevents adding the same file multiple times.
	conversionTasks sync.Map

	// A channel acting as a queue for incoming conversion requests.
	conversionQueue chan conversionJob

	// A channel acting as a semaphore to limit concurrent executions.
	concurrencyLimiter chan struct{}
)

// init runs once when the package is initialized to set up the worker queue.
func init() {
	conversionQueue = make(chan conversionJob, conversionQueueSize)
	concurrencyLimiter = make(chan struct{}, maxConcurrentConversions)
	go startConversionWorker()
}

// startConversionWorker processes jobs from the conversionQueue, respecting the concurrency limit.
func startConversionWorker() {
	for job := range conversionQueue {
		// Wait for an available slot to start a new conversion.
		concurrencyLimiter <- struct{}{}

		go func(j conversionJob) {
			// When the goroutine finishes, release the slot.
			defer func() { <-concurrencyLimiter }()
			// Perform the actual conversion.
			performConversion(j.inputFile)
		}(job)
	}
}

// ConvertToMP4 adds a video conversion job to the queue.
// It checks for duplicates in the queue or in progress.
func ConvertToMP4(inputFile database.FileData) error {
	// Atomically check if the file is already being processed and mark it as such.
	_, loaded := conversionTasks.LoadOrStore(inputFile.Md5sum, true)
	if loaded {
		return fmt.Errorf("file %s already being converted or is in queue", inputFile.OriginalFileName)
	}

	job := conversionJob{
		inputFile: inputFile,
	}

	// Try to add the job to the queue.
	select {
	case conversionQueue <- job:
		// Successfully queued.
		return nil
	default:
		// The queue is full, so we couldn't add it.
		// Remove it from our tracking map to allow it to be re-queued later.
		conversionTasks.Delete(inputFile.Md5sum)
		return fmt.Errorf("conversion queue is full")
	}
}

// performConversion contains the core logic for converting a single video file.
// It is run in a goroutine by the worker.
func performConversion(inputFile database.FileData) {
	// Ensure the file is removed from the tasks map when the conversion is done.
	defer conversionTasks.Delete(inputFile.Md5sum)

	inputFilePath := UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + inputFile.Md5sum
	outputFilePath := UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + removeExtension(inputFile.Md5sum) + ".mp4"
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		go genericUserPulse(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "input file not found: " + inputFile.OriginalFileName,
			},
		})
		return
	}
	cmd := exec.Command("ffmpeg",
		"-i", inputFilePath,
		"-c:v", "libx264",
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2", // force even dimensions
		"-preset", "veryslow",
		"-crf", "18",
		"-c:a", "copy",
		outputFilePath,
	)

	// Each conversion needs its own stderr buffer to avoid race conditions.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		go os.Remove(outputFilePath)
		go genericUserPulse(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": fmt.Sprintf("failed to convert video: %v. FFMpeg output: %s", err, stderr.String()),
			},
		})
		return
	}
	fileInfo, err := os.Stat(outputFilePath)
	if err != nil {
		go genericUserPulse(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to get output file stats: " + err.Error(),
			},
		})
		return
	}
	fileSize := fileInfo.Size()
	uniqueFileName := database.GenerateUniqueFileName(inputFile.OriginalFileName + ".mp4")
	outputMd5sum, err := md5sum(outputFilePath)
	if err != nil {
		go genericUserPulse(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to calculate MD5 checksum: " + err.Error(),
			},
		})
		return
	}
	err = os.Rename(outputFilePath, UPLOAD_DIR+string(os.PathSeparator)+"i"+string(os.PathSeparator)+outputMd5sum+".mp4")
	if err != nil {
		go genericUserPulse(inputFile.AccountToken, map[string]interface{}{
			"type": "error",
			"data": map[string]interface{}{
				"error": "failed to rename output file: " + err.Error(),
			},
		})
		return
	}
	fileData := database.FileData{
		OriginalFileName: removeExtension(inputFile.OriginalFileName) + ".mp4",
		FileDirectory:    uniqueFileName,
		AccountToken:     inputFile.AccountToken,
		FileSize:         fileSize,
		Timestamp:        time.Now().UTC().Unix(),
		Md5sum:           outputMd5sum + ".mp4",
	}

	fileData.Insert()
	go UserFilesPulse(
		FileUpdate{
			Toggle: true,
			File:   fileData,
		},
	)
	go genericUserPulse(inputFile.AccountToken, map[string]interface{}{
		"type": "convert_video_response",
		"data": map[string]interface{}{
			"file": fileData,
		},
	})
}

func HandleConversionRequest(req ConvertVideoRequest) (string, error) {
	var err error
	if req.Auth.Token == "" {
		req.Auth.Token, err = req.Auth.GetToken()
		if err != nil {
			return "", fmt.Errorf("failed to get token: %v", err)
		}
	}
	fileToConvert, err := database.GetFile(req.FileDirectory)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %v", err)
	}
	if fileToConvert.AccountToken != req.Auth.Token {
		return "", fmt.Errorf("file %s does not belong to account %s", fileToConvert.FileDirectory, req.Auth.Token)
	}
	go ConvertToMP4(fileToConvert)
	return fileToConvert.FileDirectory, nil
}
