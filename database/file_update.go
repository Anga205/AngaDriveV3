package database

import (
	"fmt"
	"strings"
	"unicode"
)

func validateFileName(filename string) error {
	if strings.TrimSpace(filename) == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if filename == "." || filename == ".." {
		return fmt.Errorf("filename is not valid")
	}
	if len(filename) > 255 {
		return fmt.Errorf("filename is too long")
	}
	for _, character := range filename {
		if character == '/' || character == '\\' || unicode.IsControl(character) {
			return fmt.Errorf("filename contains unsupported characters")
		}
	}
	return nil
}

func RenameFile(fileDirectory, newFileName string) (FileData, error) {
	if err := validateFileName(newFileName); err != nil {
		return FileData{}, err
	}

	UserFilesMutex.RLock()
	defer UserFilesMutex.RUnlock()
	CollectionFilesMutex.RLock()
	defer CollectionFilesMutex.RUnlock()
	CollectionFoldersMutex.RLock()
	defer CollectionFoldersMutex.RUnlock()
	FileCacheLock.Lock()
	defer FileCacheLock.Unlock()
	CollectionCacheLock.RLock()
	defer CollectionCacheLock.RUnlock()

	file, _, err := unsafeGetFile(fileDirectory)
	if err != nil {
		return FileData{}, err
	}
	if err := GetDB().Model(&FileData{}).
		Where("file_directory = ?", fileDirectory).
		Update("original_file_name", newFileName).Error; err != nil {
		return FileData{}, err
	}
	file.OriginalFileName = newFileName
	FileCache[fileDirectory] = file
	return file, nil
}
