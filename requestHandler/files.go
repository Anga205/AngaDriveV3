package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

func GetUserFiles(req AuthInfo) ([]database.FileData, error) {

	if req.Token == "" && (req.Email == "" || req.Password == "") {
		return nil, errors.New("missing authentication credentials")
	}

	if req.Token == "" {
		if !accounts.Authenticate(req.Email, req.Password) {
			return nil, errors.New("invalid email or password")
		}
		userInfo, _ := database.FindUserByEmail(req.Email)
		req.Token = userInfo.Token
	}
	files, err := database.GetUserFiles(req.Token)
	if err != nil {
		return nil, errors.New("failed to retrieve files")
	}
	return files, nil

}

func getExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
	}
	return ""
}

func RemoveFileIfNoClonesExist(fileToDelete database.FileData) {
	if !database.CheckForFilesWithSha256sum(fileToDelete.Sha256sum) {
		os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + fileToDelete.Sha256sum)
		ext := strings.ToLower(getExtension(fileToDelete.OriginalFileName))
		if ext == "pdf" {
			os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews" + string(os.PathSeparator) + fileToDelete.Sha256sum + ".png")
		} else {
			imageExtensions := []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"}
			for _, imgExt := range imageExtensions {
				if ext == imgExt {
					os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "image_previews" + string(os.PathSeparator) + fileToDelete.Sha256sum)
					break
				}
			}
		}
		// If the deleted file is a video and no other file shares its
		// sha256sum, remove its generated WebP preview (or its "unpreviewable"
		// marker) too.
		videoExtensions := []string{"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"}
		for _, vidExt := range videoExtensions {
			if ext == vidExt {
				os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "video_previews" + string(os.PathSeparator) + fileToDelete.Sha256sum + ".webp")
				os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "video_previews" + string(os.PathSeparator) + fileToDelete.Sha256sum + ".invalid")
				break
			}
		}
	}
}

func DeleteFile(req DeleteFileRequest) error {
	if req.Auth.Token == "" {
		if !accounts.Authenticate(req.Auth.Email, req.Auth.Password) {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Authentication failed for delete_file request\n", timestamp)
			return fmt.Errorf("authentication failed")
		}
		user, _ := database.FindUserByEmail(req.Auth.Email)
		req.Auth.Token = user.Token
	}
	fileToDelete, err := database.GetFile(req.FileDirectory)
	if err != nil {
		now := time.Now()
		timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
		fmt.Printf("[%s] Error fetching file: %v\n", timestamp, err)
		return fmt.Errorf("file not found: %v", err)
	}
	if fileToDelete.AccountToken != req.Auth.Token {
		now := time.Now()
		timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
		fmt.Printf("[%s] Unauthorized delete attempt by %s on file %s\n", timestamp, req.Auth.Email, req.FileDirectory)
		return fmt.Errorf("unauthorized delete attempt")
	}
	err = database.DeleteFile(fileToDelete, PulseCollectionSubscribers)
	if err != nil {
		now := time.Now()
		timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
		fmt.Printf("[%s] Error deleting file: %v\n", timestamp, err)
		return fmt.Errorf("error deleting file: %v", err)
	}
	go RemoveFileIfNoClonesExist(fileToDelete)
	return nil
}

// BulkDeleteFile deletes multiple files owned by the authenticated user in a
// single request. This reduces the number of websocket round-trips required to
// delete many files at once. If any individual file fails to delete (e.g. it
// does not exist or belongs to another user), the error is collected but the
// remaining files are still processed.
func BulkDeleteFile(req BulkDeleteRequest) (BulkDeleteResponse, error) {
	if req.Auth.Token == "" {
		if !accounts.Authenticate(req.Auth.Email, req.Auth.Password) {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Authentication failed for bulk_file_delete request\n", timestamp)
			return BulkDeleteResponse{}, fmt.Errorf("authentication failed")
		}
		user, _ := database.FindUserByEmail(req.Auth.Email)
		req.Auth.Token = user.Token
	}

	deleted := []string{}
	errors := []FileDeleteError{}
	for _, fileDirectory := range req.FileDirectories {
		fileToDelete, err := database.GetFile(fileDirectory)
		if err != nil {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Error fetching file %s: %v\n", timestamp, fileDirectory, err)
			errors = append(errors, FileDeleteError{FileDirectory: fileDirectory, Error: "file not found"})
			continue
		}
		if fileToDelete.AccountToken != req.Auth.Token {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Unauthorized delete attempt by %s on file %s\n", timestamp, req.Auth.Email, fileDirectory)
			errors = append(errors, FileDeleteError{FileDirectory: fileDirectory, Error: "unauthorized delete attempt"})
			continue
		}
		if err := deleteFileInternal(fileToDelete); err != nil {
			errors = append(errors, FileDeleteError{FileDirectory: fileDirectory, Error: err.Error()})
			continue
		}
		deleted = append(deleted, fileDirectory)
		go UserFilesPulse(FileUpdate{Toggle: false, File: fileToDelete})
	}

	if len(deleted) > 0 {
		go UpdateUserCount()
	}

	return BulkDeleteResponse{Deleted: deleted, Errors: errors}, nil
}

func deleteFileInternal(fileToDelete database.FileData) error {
	if err := database.DeleteFile(fileToDelete, PulseCollectionSubscribers); err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}
	ext := strings.ToLower(getExtension(fileToDelete.OriginalFileName))
	if ext == "pdf" {
		os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews" + string(os.PathSeparator) + fileToDelete.FileDirectory + ".png")
	} else {
		imageExtensions := []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"}
		for _, imgExt := range imageExtensions {
			if ext == imgExt {
				os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "image_previews" + string(os.PathSeparator) + fileToDelete.FileDirectory)
				break
			}
		}
	}
	go RemoveFileIfNoClonesExist(fileToDelete)
	return nil
}

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
