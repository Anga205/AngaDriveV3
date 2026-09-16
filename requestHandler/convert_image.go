package requestHandler

import (
	"angadrive/database"
	"fmt"
)

// HandleImageConversionRequest handles a "convert_image" websocket request.
//
// It authenticates the user, verifies ownership of the file, and enqueues an
// image-to-PNG conversion job on the runners manager. It returns immediately;
// the actual conversion runs asynchronously and the result is delivered to the
// user via a "convert_image_response" pulse.
func HandleImageConversionRequest(req ConvertImageRequest) (string, error) {
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
	if err := SubmitImageConversion(fileToConvert); err != nil {
		return "", fmt.Errorf("failed to queue conversion: %v", err)
	}
	return fileToConvert.FileDirectory, nil
}
