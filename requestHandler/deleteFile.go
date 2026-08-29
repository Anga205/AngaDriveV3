package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
	"fmt"
	"os"
	"strings"
	"time"
)

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
		// sha256sum, remove its generated GIF preview too.
		videoExtensions := []string{"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"}
		for _, vidExt := range videoExtensions {
			if ext == vidExt {
				os.Remove(globals.UPLOAD_DIR + string(os.PathSeparator) + "video_previews" + string(os.PathSeparator) + fileToDelete.Sha256sum + ".gif")
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
