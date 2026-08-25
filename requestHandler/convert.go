package requestHandler

import (
	"angadrive/database"
	"fmt"
)

// HandleConversionRequest handles a "convert_video" websocket request.
//
// It authenticates the user, verifies ownership of the file, and enqueues a
// video conversion job on the runners manager. It returns immediately; the
// actual conversion runs asynchronously and the result is delivered to the
// user via a "convert_video_response" pulse.
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
	if err := SubmitVideoConversion(fileToConvert); err != nil {
		return "", fmt.Errorf("failed to queue conversion: %v", err)
	}
	return fileToConvert.FileDirectory, nil
}
