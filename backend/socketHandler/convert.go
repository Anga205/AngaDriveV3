package socketHandler

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"service/database"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

var alreadyConverting sync.Map

func ConvertToMP4(inputFile database.FileData, mutex *sync.Mutex, conn *websocket.Conn) error {
	// Check if the file is already being converted
	_, loaded := alreadyConverting.LoadOrStore(inputFile.Md5sum, true)
	if loaded {
		mutex.Lock()
		conn.WriteJSON(OutgoingResponse{
			Type: "convert_video_response",
			Data: map[string]interface{}{
				"error": fmt.Sprintf("file %s already being converted", inputFile.OriginalFileName),
			},
		})
		mutex.Unlock()
		return fmt.Errorf("file %s already being converted", inputFile.OriginalFileName)
	}
	// Ensure the file is removed from the converting map when the function exits
	defer alreadyConverting.Delete(inputFile.Md5sum)

	inputFilePath := UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + inputFile.Md5sum
	outputFilePath := UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + removeExtension(inputFile.Md5sum) + ".mp4"
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		mutex.Lock()
		conn.WriteJSON(OutgoingResponse{
			Type: "convert_video_response",
			Data: map[string]interface{}{
				"error": "input file not found: " + inputFile.OriginalFileName,
			},
		})
		mutex.Unlock()
		return fmt.Errorf("input file not found: %w", err)
	}
	cmd := exec.Command("ffmpeg",
		"-i", inputFilePath,
		"-c:v", "libx264",
		"-preset", "veryslow",
		"-crf", "18",
		"-c:a", "copy",
		outputFilePath,
	)

	if err := cmd.Run(); err != nil {
		mutex.Lock()
		conn.WriteJSON(OutgoingResponse{
			Type: "convert_video_response",
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		mutex.Unlock()
		return fmt.Errorf("failed to convert video: %w", err)
	}
	fileInfo, err := os.Stat(outputFilePath)
	if err != nil {
		mutex.Lock()
		conn.WriteJSON(OutgoingResponse{
			Type: "convert_video_response",
			Data: map[string]interface{}{
				"error": "failed to get output file stats: " + err.Error(),
			},
		})
		mutex.Unlock()
		return fmt.Errorf("failed to get output file stats: %w", err)
	}
	fileSize := int(fileInfo.Size())
	uniqueFileName := database.GenerateUniqueFileName(inputFile.OriginalFileName)
	outputMd5sum, err := md5sum(outputFilePath)
	if err != nil {
		mutex.Lock()
		conn.WriteJSON(OutgoingResponse{
			Type: "convert_video_response",
			Data: map[string]interface{}{
				"error": "failed to calculate MD5 checksum: " + err.Error(),
			},
		})
		mutex.Unlock()
		return fmt.Errorf("failed to calculate MD5 checksum: %w", err)
	}
	err = os.Rename(outputFilePath, UPLOAD_DIR+string(os.PathSeparator)+"i"+string(os.PathSeparator)+outputMd5sum+".mp4")
	if err != nil {
		mutex.Lock()
		conn.WriteJSON(OutgoingResponse{
			Type: "convert_video_response",
			Data: map[string]interface{}{
				"error": "failed to rename output file: " + err.Error(),
			},
		})
		mutex.Unlock()
		return fmt.Errorf("failed to rename output file: %w", err)
	}
	fileData := database.FileData{
		OriginalFileName: removeExtension(inputFile.OriginalFileName) + ".mp4",
		FileDirectory:    uniqueFileName,
		AccountToken:     inputFile.AccountToken,
		FileSize:         fileSize,
		Timestamp:        time.Now().UTC().Unix(),
		Md5sum:           outputMd5sum + ".mp4",
	}
	database.InsertNewFile(fileData)
	go UserFilesPulse(
		FileUpdate{
			Toggle: true,
			File:   fileData,
		},
	)
	mutex.Lock()
	conn.WriteJSON(OutgoingResponse{
		Type: "convert_video_response",
		Data: map[string]interface{}{
			"file": fileData,
		},
	})
	mutex.Unlock()
	return nil
}
