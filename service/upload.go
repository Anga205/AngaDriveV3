package service

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/protocol"
	"angadrive/storage"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FinalizeUpload assembles the uploaded chunks into a single file, records it in
// the database, optionally adds it to a collection, and broadcasts the update.
// It returns the created file metadata.
func FinalizeUpload(accountToken, uploadID, originalFileName, collectionID string, totalChunks int) (database.FileData, error) {
	uploadPath := filepath.Join(storage.ChunksDir(), uploadID)
	if missing := MissingChunks(uploadPath, totalChunks); len(missing) > 0 {
		return database.FileData{}, &MissingChunksError{Missing: missing}
	}

	finalDestDir := storage.FilesDir()
	if err := os.MkdirAll(finalDestDir, os.ModePerm); err != nil {
		return database.FileData{}, fmt.Errorf("failed to create destination directory")
	}
	tempFile, err := os.CreateTemp(finalDestDir, "upload-*.tmp")
	if err != nil {
		return database.FileData{}, fmt.Errorf("failed to create temporary file")
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return database.FileData{}, fmt.Errorf("failed to open chunk %d: %v", i, err)
		}
		_, err = io.Copy(tempFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			return database.FileData{}, fmt.Errorf("failed to copy chunk %d: %v", i, err)
		}
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return database.FileData{}, fmt.Errorf("failed to rewind temporary file")
	}
	sha256sum, err := storage.SHA256File(tempFile.Name())
	if err != nil {
		return database.FileData{}, fmt.Errorf("failed to calculate SHA-256 hash")
	}
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return database.FileData{}, fmt.Errorf("failed to get final file stats")
	}

	finalFilePath := filepath.Join(finalDestDir, sha256sum+filepath.Ext(originalFileName))
	tempFile.Close()
	if err := os.Rename(tempFile.Name(), finalFilePath); err != nil {
		return database.FileData{}, fmt.Errorf("failed to rename temporary file")
	}

	fileData := database.FileData{
		OriginalFileName: originalFileName,
		FileDirectory:    database.GenerateUniqueFileName(originalFileName),
		AccountToken:     accountToken,
		FileSize:         fileInfo.Size(),
		Timestamp:        time.Now().Unix(),
		Sha256sum:        sha256sum + filepath.Ext(originalFileName),
	}
	if err := fileData.Insert(); err != nil {
		os.Remove(finalFilePath)
		return database.FileData{}, fmt.Errorf("failed to insert file metadata: %v", err)
	}

	if collectionID != "" {
		_, err := AddFileToCollection(protocol.AddFileToCollectionRequest{
			CollectionID:  collectionID,
			FileDirectory: fileData.FileDirectory,
			Auth:          protocol.AuthInfo{Token: accountToken},
		})
		if err != nil {
			fmt.Printf("Warning: Failed to add file %s to collection %s: %v\n", fileData.FileDirectory, collectionID, err)
		}
	}

	os.RemoveAll(uploadPath)

	go hub.PublishFileUpdate(protocol.FileUpdate{Toggle: true, File: fileData})
	go hub.UpdateUserCount()
	return fileData, nil
}

// MissingChunksError reports which chunk indices are absent.
type MissingChunksError struct {
	Missing []int
}

func (e *MissingChunksError) Error() string {
	return fmt.Sprintf("missing %d chunks", len(e.Missing))
}

// MissingChunks returns the indices of chunks that have not been uploaded.
func MissingChunks(uploadPath string, totalChunks int) []int {
	var missing []int
	for i := 0; i < totalChunks; i++ {
		if _, err := os.Stat(filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))); os.IsNotExist(err) {
			missing = append(missing, i)
		}
	}
	return missing
}
