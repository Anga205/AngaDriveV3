package socketHandler

import (
	"fmt"
	"os"
	"service/accounts"
	"service/database"
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

func RemoveFile(md5sum string) {
	if !database.CheckForFilesWithMd5sum(md5sum) {
		os.Remove(UPLOAD_DIR + string(os.PathSeparator) + "i" + string(os.PathSeparator) + md5sum)
		if getExtension(md5sum) == "pdf" {
			os.Remove(UPLOAD_DIR + string(os.PathSeparator) + "pdf_previews" + string(os.PathSeparator) + md5sum + ".png")
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
	err = database.DeleteFile(fileToDelete)
	if err != nil {
		now := time.Now()
		timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
		fmt.Printf("[%s] Error deleting file: %v\n", timestamp, err)
		return fmt.Errorf("error deleting file: %v", err)
	}
	go RemoveFile(fileToDelete.Md5sum)
	return nil
}
