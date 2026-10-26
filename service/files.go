package service

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/protocol"
	"angadrive/storage"
	"fmt"
	"os"
	"strings"
	"time"
)

// GetUserFiles returns all files owned by the authenticated user.
func GetUserFiles(req protocol.AuthInfo) ([]database.FileData, error) {
	token, err := Authenticate(req)
	if err != nil {
		return nil, err
	}
	files, err := database.GetUserFiles(token)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve files")
	}
	return files, nil
}

// DeleteFile deletes a single file owned by the authenticated user and returns
// the deleted file's metadata.
func DeleteFile(req protocol.DeleteFileRequest) (database.FileData, error) {
	token, err := AuthenticateOrToken(req.Auth)
	if err != nil {
		return database.FileData{}, err
	}
	file, err := database.GetFile(req.FileDirectory)
	if err != nil {
		return database.FileData{}, fmt.Errorf("file not found: %v", err)
	}
	if file.AccountToken != token {
		return database.FileData{}, fmt.Errorf("unauthorized delete attempt")
	}
	if err := deleteFileInternal(file); err != nil {
		return database.FileData{}, err
	}
	go hub.PublishFileUpdate(protocol.FileUpdate{Toggle: false, File: file})
	go hub.UpdateUserCount()
	return file, nil
}

// deleteFileInternal removes a file from the database and cleans up its
// generated previews and physical blob when no clones remain.
func deleteFileInternal(file database.FileData) error {
	affected, err := database.DeleteFile(file)
	if err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}
	for _, collection := range affected {
		go hub.PublishCollection(collection)
	}
	removePreviewFiles(file)
	go RemoveFileIfNoClonesExist(file)
	return nil
}

// BulkDeleteFile deletes multiple files, collecting per-file errors.
func BulkDeleteFile(req protocol.BulkDeleteRequest) (protocol.BulkDeleteResponse, error) {
	token, err := AuthenticateOrToken(req.Auth)
	if err != nil {
		return protocol.BulkDeleteResponse{}, err
	}

	deleted := []string{}
	errs := []protocol.FileDeleteError{}
	for _, fileDirectory := range req.FileDirectories {
		file, err := database.GetFile(fileDirectory)
		if err != nil {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: "file not found"})
			continue
		}
		if file.AccountToken != token {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: "unauthorized delete attempt"})
			continue
		}
		if err := deleteFileInternal(file); err != nil {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: err.Error()})
			continue
		}
		deleted = append(deleted, fileDirectory)
		go hub.PublishFileUpdate(protocol.FileUpdate{Toggle: false, File: file})
	}
	if len(deleted) > 0 {
		go hub.UpdateUserCount()
	}
	return protocol.BulkDeleteResponse{Deleted: deleted, Errors: errs}, nil
}

// DuplicateFile creates a copy of a file owned by the authenticated user.
func DuplicateFile(req protocol.DuplicateFileRequest) (database.FileData, error) {
	token, err := Authenticate(req.Auth)
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
	duplicate := newDuplicate(original)
	if err := duplicate.Insert(); err != nil {
		return database.FileData{}, fmt.Errorf("failed to duplicate file: %v", err)
	}
	go hub.PublishFileUpdate(protocol.FileUpdate{Toggle: true, File: duplicate})
	return duplicate, nil
}

// BulkDuplicateFile duplicates multiple files, collecting per-file errors.
func BulkDuplicateFile(req protocol.BulkDuplicateRequest) (protocol.BulkDuplicateResponse, error) {
	token, err := AuthenticateOrToken(req.Auth)
	if err != nil {
		return protocol.BulkDuplicateResponse{}, err
	}

	duplicated := []string{}
	errs := []protocol.FileDeleteError{}
	for _, fileDirectory := range req.FileDirectories {
		original, err := database.GetFile(fileDirectory)
		if err != nil {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: "file not found"})
			continue
		}
		if original.AccountToken != token {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: "unauthorized duplicate attempt"})
			continue
		}
		duplicate := newDuplicate(original)
		if err := duplicate.Insert(); err != nil {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: fmt.Sprintf("failed to duplicate file: %v", err)})
			continue
		}
		duplicated = append(duplicated, fileDirectory)
		go hub.PublishFileUpdate(protocol.FileUpdate{Toggle: true, File: duplicate})
	}
	if len(duplicated) > 0 {
		go hub.UpdateUserCount()
	}
	return protocol.BulkDuplicateResponse{Duplicated: duplicated, Errors: errs}, nil
}

func newDuplicate(original database.FileData) database.FileData {
	name := "Copy of " + original.OriginalFileName
	return database.FileData{
		OriginalFileName: name,
		FileDirectory:    database.GenerateUniqueFileName(name),
		AccountToken:     original.AccountToken,
		FileSize:         original.FileSize,
		Timestamp:        time.Now().Unix(),
		Sha256sum:        original.Sha256sum,
	}
}

// RenameFile renames a file owned by the authenticated user.
func RenameFile(req protocol.RenameFileRequest) (database.FileData, error) {
	token, err := Authenticate(req.Auth)
	if err != nil {
		return database.FileData{}, err
	}
	file, err := database.GetFile(req.FileDirectory)
	if err != nil {
		return database.FileData{}, fmt.Errorf("file not found")
	}
	if file.AccountToken != token {
		return database.FileData{}, fmt.Errorf("unauthorized rename attempt")
	}
	updated, err := database.RenameFile(req.FileDirectory, req.NewFileName)
	if err != nil {
		return database.FileData{}, err
	}
	go hub.PublishFileUpdate(protocol.FileUpdate{Toggle: true, Replace: true, File: updated})
	go hub.PublishFileToSubscribers(updated)
	return updated, nil
}

// RemoveFileIfNoClonesExist deletes the physical blob and generated previews
// when no other file row references the same content hash.
func RemoveFileIfNoClonesExist(file database.FileData) {
	if database.CheckForFilesWithSha256sum(file.Sha256sum) {
		return
	}
	os.Remove(storage.FilePath(file.Sha256sum))
	removePreviewFiles(file)
}

// removePreviewFiles deletes any generated previews for a file.
func removePreviewFiles(file database.FileData) {
	ext := strings.ToLower(storage.Extension(file.OriginalFileName))
	switch {
	case ext == "pdf":
		os.Remove(storage.PDFPreviewPath(file.Sha256sum))
	case isImageExt(ext):
		os.Remove(storage.ImagePreviewPath(file.Sha256sum))
	case isVideoExt(ext):
		os.Remove(storage.VideoPreviewPath(file.Sha256sum))
	}
}

func isImageExt(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff":
		return true
	}
	return false
}

func isVideoExt(ext string) bool {
	switch ext {
	case "mp4", "mkv", "avi", "mov", "wmv", "flv", "webm":
		return true
	}
	return false
}
