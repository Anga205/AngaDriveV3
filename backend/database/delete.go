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

func (collection Collection) unsafeDelete() error {
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeDelete")
	}
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFilesMutex before calling unsafeDelete")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFoldersMutex before calling unsafeDelete")
	}
	if UserCollectionsMutex.TryLock() {
		defer UserCollectionsMutex.Unlock()
		return fmt.Errorf("please Lock UserCollectionsMutex before calling unsafeDelete")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeDelete")
	}
	db := GetDB()
	result := db.Delete(&Collection{}, collection)
	if result.Error != nil {
		return result.Error
	}
	var collectionsContainingThisCollection []Collection
	db.Where("collections LIKE ?", "%"+collection.ID+"%").Find(&collectionsContainingThisCollection)
	for _, c := range collectionsContainingThisCollection {
		c.unsafeRemoveFolder(collection.ID)
	}
	go func(dependantID string) {
		var dependantCollections []Collection
		db.Where("dependant = ?", dependantID).Find(&dependantCollections)

		for _, dependantCollection := range dependantCollections {
			go dependantCollection.Delete()
		}
	}(collection.ID)
	delete(CollectionCache, collection.ID) // We assume CollectionCacheLock is already locked
	for user := range UserCollections {
		if UserCollections[user] != nil {
			UserCollections[user].Remove(collection.ID) // We assume UserCollectionsMutex is already locked
		}
	}
	delete(CollectionFiles, collection.ID)   // We assume CollectionFilesMutex is already locked
	delete(CollectionFolders, collection.ID) // We assume CollectionFoldersMutex is already locked
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection Collection) Delete() error {
	CollectionCacheLock.Lock()
	defer CollectionCacheLock.Unlock()
	CollectionFilesMutex.Lock()
	defer CollectionFilesMutex.Unlock()
	CollectionFoldersMutex.Lock()
	defer CollectionFoldersMutex.Unlock()
	UserCollectionsMutex.RLock()
	defer UserCollectionsMutex.RUnlock()
	FileCacheLock.RLock()
	defer FileCacheLock.RUnlock()
	return collection.unsafeDelete()
}

func unsafeDeleteFile(file FileData, collectionPulser func(collection Collection)) error {
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Lock FileCacheLock before calling unsafeDeleteFile")
	}
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFilesMutex before calling unsafeDeleteFile")
	}
	if UserFilesMutex.TryLock() {
		defer UserFilesMutex.Unlock()
		return fmt.Errorf("please Read-Lock UserFilesMutex before calling unsafeDeleteFile")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeDeleteFile")
	}
	db := GetDB()
	result := db.Delete(&FileData{}, file)
	if result.Error != nil {
		return result.Error
	}
	var collections []Collection
	db.Where("Files LIKE ?", "%"+file.FileDirectory+"%").Find(&collections)
	updatedCollections := make(map[string]bool)
	for _, collection := range collections {
		collection.Files = removeFile(collection.Files, file.FileDirectory)
		collection.Size = unsafeCalculateCollectionSize(collection, make(map[string]bool), make(map[string]bool))
		updatedCollections[collection.ID] = true
		if err := db.Save(&collection).Error; err != nil {
			return fmt.Errorf("failed to update collection %s: %v", collection.Name, err)
		} else {
			if cachedCollection, ok := CollectionCache[collection.ID]; ok {
				if cachedCollection.Files != collection.Files {
					CollectionCache[collection.ID] = collection // Update the cache if the files have changed
				}
			}
			go collectionPulser(collection)
			unsafeUpdateParentCollectionSizes(collection.ID, &updatedCollections)
		}
	}
	for _, fileSet := range UserFiles {
		if fileSet != nil {
			fileSet.Remove(file.FileDirectory)
		}
	}
	delete(FileCache, file.FileDirectory)
	for _, fileSet := range CollectionFiles {
		if fileSet != nil {
			fileSet.Remove(file.FileDirectory)
		}
	}
	return nil
}

func DeleteFile(file FileData, collectionPulser func(collection Collection)) error {
	FileCacheLock.Lock()
	defer FileCacheLock.Unlock()
	CollectionFilesMutex.RLock()
	defer CollectionFilesMutex.RUnlock()
	UserFilesMutex.RLock()
	defer UserFilesMutex.RUnlock()
	CollectionCacheLock.Lock()
	defer CollectionCacheLock.Unlock()
	return unsafeDeleteFile(file, collectionPulser)
}
