package service

import (
	"angadrive/database"
	"angadrive/protocol"
	"angadrive/runners"
	"fmt"
)

// HandleConversionRequest authenticates, verifies ownership, and enqueues a
// video conversion job.
func HandleConversionRequest(req protocol.ConvertVideoRequest) (string, error) {
	token, err := AuthenticateOrToken(req.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}
	file, err := database.GetFile(req.FileDirectory)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %v", err)
	}
	if file.AccountToken != token {
		return "", fmt.Errorf("file %s does not belong to account %s", file.FileDirectory, token)
	}
	if err := SubmitVideoConversion(file); err != nil {
		return "", fmt.Errorf("failed to queue conversion: %v", err)
	}
	return file.FileDirectory, nil
}

// HandleImageConversionRequest authenticates, verifies ownership, and enqueues
// an image conversion job.
func HandleImageConversionRequest(req protocol.ConvertImageRequest) (string, error) {
	token, err := AuthenticateOrToken(req.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}
	file, err := database.GetFile(req.FileDirectory)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %v", err)
	}
	if file.AccountToken != token {
		return "", fmt.Errorf("file %s does not belong to account %s", file.FileDirectory, token)
	}
	if err := SubmitImageConversion(file, req.TargetFormat); err != nil {
		return "", fmt.Errorf("failed to queue conversion: %v", err)
	}
	return file.FileDirectory, nil
}

// SubmitVideoConversion enqueues a video conversion job.
func SubmitVideoConversion(file database.FileData) error {
	return runners.Submit("video", runners.VideoJob{File: file})
}

// SubmitImageConversion enqueues an image conversion job.
func SubmitImageConversion(file database.FileData, targetFormat string) error {
	return runners.Submit("image", runners.ImageJob{File: file, TargetFormat: targetFormat})
}

// SubmitVideoPreview enqueues a video preview generation job.
func SubmitVideoPreview(file database.FileData) error {
	return runners.Submit("video_preview", runners.VideoPreviewJob{File: file})
}
