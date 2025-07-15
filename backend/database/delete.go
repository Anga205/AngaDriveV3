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

	var collectionsContainingThisCollection []Collection
	db.Where("collections LIKE ?", "%"+collection.ID+"%").Find(&collectionsContainingThisCollection)
	for _, c := range collectionsContainingThisCollection {
		c.RemoveFolder(collection.ID)
	}

	go func(dependantID string) { // Delete the collections that depend on this collection
		var dependantCollections []Collection
		db.Where("dependant = ?", dependantID).Find(&dependantCollections)

		for _, dependantCollection := range dependantCollections {
			dependantCollection.Delete()
		}
	}(collection.ID)

	go func() {
		func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			delete(CollectionCache, collection.ID)
		}()

		func() {
			UserCollectionsMutex.Lock()
			defer UserCollectionsMutex.Unlock()
			for user := range UserCollections {
				if UserCollections[user] != nil {
					UserCollections[user].Remove(collection.ID)
				}
			}
		}()

		func() {
			CollectionFilesMutex.Lock()
			defer CollectionFilesMutex.Unlock()
			delete(CollectionFiles, collection.ID)
		}()

		func() {
			CollectionFoldersMutex.Lock()
			defer CollectionFoldersMutex.Unlock()
			delete(CollectionFolders, collection.ID)
		}()
		updateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	}()
	return nil
}

func DeleteFile(file FileData, collectionPulser func(collection Collection)) error {

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
	updatedCollections := make(map[string]bool)
	for _, collection := range collections {
		collection.Files = removeFile(collection.Files, file.FileDirectory)
		collection.Size = calculateCollectionSize(collection, make(map[string]bool), make(map[string]bool))
		updatedCollections[collection.ID] = true
		if err := db.Save(&collection).Error; err != nil {
			return fmt.Errorf("failed to update collection %s: %v", collection.Name, err)
		} else {
			go collectionPulser(collection)
			updateParentCollectionSizes(collection.ID, &updatedCollections)
		}
	}

	go func(file FileData) { // Remove from UserFiles Cache
		UserFilesMutex.Lock()
		fmt.Println("Removing file from UserFiles cache:", file.FileDirectory)
		for _, fileSet := range UserFiles {
			if fileSet != nil {
				fileSet.Remove(file.FileDirectory)
			}
		}
		fmt.Println("File removed from UserFiles cache:", file.FileDirectory)
		UserFilesMutex.Unlock()
	}(file)

	go func(file FileData) { // Remove from FileCache
		fmt.Println("Removing file from FileCache:", file.FileDirectory)
		FileCacheLock.Lock()
		delete(FileCache, file.FileDirectory)
		FileCacheLock.Unlock()
		fmt.Println("File removed from FileCache:", file.FileDirectory)
	}(file)

	go func(file FileData) { // Remove from CollectionFiles Cache and then collection cache
		CollectionFilesMutex.Lock()
		for _, fileSet := range CollectionFiles {
			if fileSet != nil {
				fileSet.Remove(file.FileDirectory)
			}
		}
		CollectionFilesMutex.Unlock()

		var CollectionsToUpdate []Collection
		CollectionCacheLock.RLock()
		for _, collection := range CollectionCache {
			newFiles := removeFile(collection.Files, file.FileDirectory)
			if newFiles != collection.Files {
				collection.Files = newFiles
				CollectionsToUpdate = append(CollectionsToUpdate, collection)
			}
		}
		CollectionCacheLock.RUnlock()

		updatedCollections := make(map[string]bool)
		for _, collection := range CollectionsToUpdate {
			collection.Files = removeFile(collection.Files, file.FileDirectory)
			collection.Size = calculateCollectionSize(collection, make(map[string]bool), make(map[string]bool))
			updatedCollections[collection.ID] = true
			updateParentCollectionSizes(collection.ID, &updatedCollections)
			CollectionCacheLock.Lock()
			CollectionCache[collection.ID] = collection
			CollectionCacheLock.Unlock()
		}

	}(file)

	return nil
}
