package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"fmt"
	"time"
)

func BulkDuplicateFile(req BulkDuplicateRequest) (BulkDuplicateResponse, error) {
	if req.Auth.Token == "" {
		if !accounts.Authenticate(req.Auth.Email, req.Auth.Password) {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Authentication failed for bulk_file_duplicate request\n", timestamp)
			return BulkDuplicateResponse{}, fmt.Errorf("authentication failed")
		}
		user, _ := database.FindUserByEmail(req.Auth.Email)
		req.Auth.Token = user.Token
	}

	duplicated := []string{}
	errors := []FileDeleteError{}
	for _, fileDirectory := range req.FileDirectories {
		original, err := database.GetFile(fileDirectory)
		if err != nil {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Error fetching file %s: %v\n", timestamp, fileDirectory, err)
			errors = append(errors, FileDeleteError{FileDirectory: fileDirectory, Error: "file not found"})
			continue
		}
		if original.AccountToken != req.Auth.Token {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Unauthorized duplicate attempt by %s on file %s\n", timestamp, req.Auth.Email, fileDirectory)
			errors = append(errors, FileDeleteError{FileDirectory: fileDirectory, Error: "unauthorized duplicate attempt"})
			continue
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
			errors = append(errors, FileDeleteError{FileDirectory: fileDirectory, Error: fmt.Sprintf("failed to duplicate file: %v", err)})
			continue
		}
		duplicated = append(duplicated, fileDirectory)
		go UserFilesPulse(FileUpdate{Toggle: true, File: duplicate})
	}

	if len(duplicated) > 0 {
		go UpdateUserCount()
	}

	return BulkDuplicateResponse{Duplicated: duplicated, Errors: errors}, nil
}
