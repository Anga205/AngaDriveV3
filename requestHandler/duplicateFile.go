package requestHandler

import (
	"angadrive/database"
	"fmt"
	"time"
)

func DuplicateFile(req DuplicateFileRequest) (database.FileData, error) {
	token, err := req.Auth.GetToken()
	if err != nil {
		return database.FileData{}, err
	}

	original, err := database.GetFile(req.FileDirectory)
	if err != nil {
		return database.FileData{}, fmt.Errorf("file not found")
	}
	if original.AccountToken != token {
		return database.FileData{}, fmt.Errorf("unauthorized duplicate attempt")
	}

	duplicate := database.FileData{
		OriginalFileName: "Copy of " + original.OriginalFileName,
		FileDirectory:    database.GenerateUniqueFileName("Copy of " + original.OriginalFileName),
		AccountToken:     original.AccountToken,
		FileSize:         original.FileSize,
		Timestamp:        time.Now().Unix(),
		Sha256sum:        original.Sha256sum,
	}
	if err := duplicate.Insert(); err != nil {
		return database.FileData{}, fmt.Errorf("failed to duplicate file: %v", err)
	}
	return duplicate, nil
}
