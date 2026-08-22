package database

import (
	"fmt"
)

func (collection Collection) unsafeDelete() error {
	if UserCollectionsMutex.TryLock() {
		defer UserCollectionsMutex.Unlock()
		return fmt.Errorf("please Lock UserCollectionsMutex before calling unsafeDelete")
	}
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFilesMutex before calling unsafeDelete")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFoldersMutex before calling unsafeDelete")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeDelete")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeDelete")
	}
	db := GetDB()

	// Capture direct parents before removing edges so we can update their
	// sizes after the relationships are gone.
	var parentMappings []CollectionChild
	db.Where("child_collection_id = ?", collection.ID).Find(&parentMappings)

	result := db.Delete(&Collection{}, collection)
	if result.Error != nil {
		return result.Error
	}
	// Delete all relationship edges involving this collection.
	if err := db.Where("collection_id = ?", collection.ID).Delete(&CollectionFile{}).Error; err != nil {
		return err
	}
	if err := db.Where("parent_collection_id = ? OR child_collection_id = ?", collection.ID, collection.ID).
		Delete(&CollectionChild{}).Error; err != nil {
		return err
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

	// Remove this collection from its parents' RAM sets and update their sizes.
	alreadyUpdated := map[string]bool{collection.ID: true}
	for _, m := range parentMappings {
		if CollectionFolders[m.ParentCollectionID] != nil {
			CollectionFolders[m.ParentCollectionID].Remove(collection.ID)
		}
		parent, _, err := unsafeGetCollection(m.ParentCollectionID)
		if err != nil {
			continue
		}
		parent.Size = unsafeCalculateCollectionSize(parent, make(map[string]bool), make(map[string]bool))
		if err := db.Model(&Collection{}).Where("id = ?", parent.ID).Update("size", parent.Size).Error; err != nil {
			fmt.Printf("Error updating collection size for %s: %v\n", parent.ID, err)
			continue
		}
		CollectionCache[parent.ID] = parent
		alreadyUpdated[parent.ID] = true
		unsafeUpdateParentCollectionSizes(parent.ID, &alreadyUpdated)
	}
	return nil
}

func (collection Collection) Delete() error {
	UserCollectionsMutex.RLock()
	defer UserCollectionsMutex.RUnlock()
	CollectionFilesMutex.Lock()
	defer CollectionFilesMutex.Unlock()
	CollectionFoldersMutex.Lock()
	defer CollectionFoldersMutex.Unlock()
	FileCacheLock.RLock()
	defer FileCacheLock.RUnlock()
	CollectionCacheLock.Lock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeDelete()
}

func unsafeDeleteFile(file FileData, collectionPulser func(collection Collection)) error {
	if UserFilesMutex.TryLock() {
		defer UserFilesMutex.Unlock()
		return fmt.Errorf("please Read-Lock UserFilesMutex before calling unsafeDeleteFile")
	}
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFilesMutex before calling unsafeDeleteFile")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFoldersMutex before calling unsafeDeleteFile")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Lock FileCacheLock before calling unsafeDeleteFile")
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
	var mappings []CollectionFile
	db.Where("file_id = ?", file.FileDirectory).Find(&mappings)
	// Remove the file from each collection's RAM set, then delete the edges.
	updatedCollections := make(map[string]bool)
	for _, m := range mappings {
		if CollectionFiles[m.CollectionID] != nil {
			CollectionFiles[m.CollectionID].Remove(file.FileDirectory)
		}
		collection, _, err := unsafeGetCollection(m.CollectionID)
		if err != nil {
			continue
		}
		collection.Size = unsafeCalculateCollectionSize(collection, make(map[string]bool), make(map[string]bool))
		updatedCollections[collection.ID] = true
		if err := db.Model(&Collection{}).Where("id = ?", collection.ID).Update("size", collection.Size).Error; err != nil {
			return fmt.Errorf("failed to update collection %s: %v", collection.Name, err)
		}
		CollectionCache[collection.ID] = collection
		go collectionPulser(collection)
		unsafeUpdateParentCollectionSizes(collection.ID, &updatedCollections)
	}
	// Delete all collection_files edges referring to this file.
	if err := db.Where("file_id = ?", file.FileDirectory).Delete(&CollectionFile{}).Error; err != nil {
		return err
	}
	for _, fileSet := range UserFiles {
		if fileSet != nil {
			fileSet.Remove(file.FileDirectory)
		}
	}
	delete(FileCache, file.FileDirectory)
	return nil
}

func DeleteFile(file FileData, collectionPulser func(collection Collection)) error {
	UserFilesMutex.RLock()
	defer UserFilesMutex.RUnlock()
	CollectionFilesMutex.RLock()
	defer CollectionFilesMutex.RUnlock()
	CollectionFoldersMutex.RLock()
	defer CollectionFoldersMutex.RUnlock()
	FileCacheLock.Lock()
	defer FileCacheLock.Unlock()
	CollectionCacheLock.Lock()
	defer CollectionCacheLock.Unlock()
	return unsafeDeleteFile(file, collectionPulser)
}

func (account Account) Delete() error {
	UserFilesMutex.Lock()
	defer UserFilesMutex.Unlock()
	UserCollectionsMutex.Lock()
	defer UserCollectionsMutex.Unlock()
	UserAccountsByEmailMutex.Lock()
	defer UserAccountsByEmailMutex.Unlock()
	UserAccountsByTokenMutex.Lock()
	defer UserAccountsByTokenMutex.Unlock()
	db := GetDB()
	result := db.Where("token = ?", account.Token).Delete(&Account{})
	if result.Error != nil {
		return result.Error
	}
	delete(UserFiles, account.Token)
	delete(UserCollections, account.Token)
	delete(UserAccountsByEmail, account.Email)
	delete(UserAccountsByToken, account.Token)
	return nil
}
