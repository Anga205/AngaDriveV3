package database

import (
	"fmt"
	"strings"
)

func removeFile(fileList string, file string) string {
	// Split fileList into slice
	files := strings.Split(fileList, ",")

	// Prepare slice to hold filtered results
	var filtered []string

	// Iterate and exclude the file
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" && f != file {
			filtered = append(filtered, f)
		}
	}

	// Join filtered results back into comma-separated string
	return strings.Join(filtered, ",")
}

func (collection Collection) Delete() error {
	db := GetDB()
	result := db.Delete(&Collection{}, collection)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}

	go func(dependantID string) {
		var dependantCollections []Collection
		db.Where("dependant = ?", dependantID).Find(&dependantCollections)

		for _, dependantCollection := range dependantCollections {
			dependantCollection.Delete()
		}
	}(collection.ID)

	go func() {
		CollectionCacheLock.Lock()
		defer CollectionCacheLock.Unlock()
		delete(CollectionCache, collection.ID)
	}()

	go func() {
		UserCollectionsMutex.Lock()
		defer UserCollectionsMutex.Unlock()
		delete(UserCollections, collection.ID)
	}()

	go func() {
		CollectionFilesMutex.Lock()
		defer CollectionFilesMutex.Unlock()
		delete(CollectionFiles, collection.ID)
	}()

	go func() {
		CollectionFoldersMutex.Lock()
		defer CollectionFoldersMutex.Unlock()
		delete(CollectionFolders, collection.ID)
	}()

	return nil
}

func DeleteFile(file FileData) error {

	go func() { // Remove from UserFiles Cache
		UserFilesMutex.Lock()
		for _, fileSet := range UserFiles {
			if fileSet != nil {
				fileSet.Remove(file.FileDirectory)
			}
		}
		UserFilesMutex.Unlock()
	}()

	go func() { // Remove from CollectionFiles Cache
		CollectionFilesMutex.Lock()
		for _, fileSet := range CollectionFiles {
			if fileSet != nil {
				fileSet.Remove(file.FileDirectory)
			}
		}
		CollectionFilesMutex.Unlock()
	}()

	go func() { // Remove from FileCache
		FileCacheLock.Lock()
		delete(FileCache, file.FileDirectory)
		FileCacheLock.Unlock()
	}()

	go func() { // remove from collection cache
		CollectionCacheLock.Lock()
		for key, collection := range CollectionCache {
			collection.Files = removeFile(collection.Files, file.FileDirectory)
			CollectionCache[key] = collection
		}
		CollectionCacheLock.Unlock()
	}()

	// Delete from database
	db := GetDB()
	result := db.Delete(&FileData{}, file)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("file not found: %v", file)
	}
	// Delete file from collections
	var collections []Collection
	db.Where("Files LIKE ?", "%"+file.FileDirectory+"%").Find(&collections)
	for _, collection := range collections {
		collection.Files = removeFile(collection.Files, file.FileDirectory)
		if err := db.Save(&collection).Error; err != nil {
			return fmt.Errorf("failed to update collection %s: %v", collection.Name, err)
		}
	}
	return nil
}
