package database

import (
	"fmt"
)

func (oldInfo Account) Update(newInfo Account) error { // this assumes token can not change
	if oldInfo.Token != newInfo.Token {
		return fmt.Errorf("token cannot be changed, old: %s, new: %s", oldInfo.Token, newInfo.Token)
	}
	db := GetDB()
	err := db.Model(&Account{}).Where("token = ?", oldInfo.Token).Updates(newInfo).Error
	if err != nil {
		return err
	}
	go func() {
		UserAccountsByEmailMutex.Lock()
		defer UserAccountsByEmailMutex.Unlock()
		if oldInfo.Email != newInfo.Email {
			delete(UserAccountsByEmail, oldInfo.Email)
		}
		UserAccountsByEmail[newInfo.Email] = newInfo
	}()
	go func() {
		UserAccountsByTokenMutex.Lock()
		defer UserAccountsByTokenMutex.Unlock()
		UserAccountsByToken[newInfo.Token] = newInfo
	}()
	return nil
}

func (collection *Collection) unsafeAddFolder(folder string) error {
	var err error
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFilesMutex before calling unsafeAddFolder")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFoldersMutex before calling unsafeAddFolder")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeAddFolder")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeAddFolder")
	}
	if folder == collection.ID {
		return fmt.Errorf("a collection cannot be its own child")
	}
	// CollectionCacheLock is already locked
	// we're doing this to ensure that the collection still exists and is up-to-date by the time the locks were acquired and the function was called
	*collection, _, err = unsafeGetCollection(collection.ID)
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	db := GetDB()
	// Persist the relational mapping; ignore duplicates (ON CONFLICT semantics via primary key).
	if err = db.Create(&CollectionChild{
		ParentCollectionID: collection.ID,
		ChildCollectionID:  folder,
	}).Error; err != nil {
		if !isDuplicateKeyError(err) {
			return err
		}
	}
	// Update RAM before recalculating size so the new child is reflected.
	if _, ok := CollectionFolders[collection.ID]; ok {
		CollectionFolders[collection.ID].Add(folder)
	}
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	if err = db.Model(&Collection{}).Where("id = ?", collection.ID).Update("size", collection.Size).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) AddFolder(folder string) error {
	CollectionFilesMutex.RLock()
	CollectionFoldersMutex.Lock()
	FileCacheLock.RLock()
	CollectionCacheLock.Lock()
	defer CollectionFilesMutex.RUnlock()
	defer CollectionFoldersMutex.Unlock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeAddFolder(folder)
}

func (collection *Collection) unsafeRemoveFolder(folder string) error {
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFilesMutex before calling unsafeRemoveFolder")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFoldersMutex before calling unsafeRemoveFolder")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeRemoveFolder")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeRemoveFolder")
	}
	var err error
	*collection, _, err = unsafeGetCollection(collection.ID) // CollectionCacheLock is already locked
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	db := GetDB()
	if err := db.Where("parent_collection_id = ? AND child_collection_id = ?", collection.ID, folder).
		Delete(&CollectionChild{}).Error; err != nil {
		return err
	}
	// Update RAM before recalculating size so the removed child is reflected.
	if _, ok := CollectionFolders[collection.ID]; ok {
		CollectionFolders[collection.ID].Remove(folder)
	}
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	if err := db.Model(&Collection{}).Where("id = ?", collection.ID).Update("size", collection.Size).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) RemoveFolder(folder string) error {
	CollectionFilesMutex.RLock()
	CollectionFoldersMutex.Lock()
	FileCacheLock.RLock()
	CollectionCacheLock.Lock()
	defer CollectionFilesMutex.RUnlock()
	defer CollectionFoldersMutex.Unlock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeRemoveFolder(folder)
}

func (collection *Collection) unsafeAddFile(fileDirectory string) error {
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFilesMutex before calling unsafeAddFile")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFoldersMutex before calling unsafeAddFile")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeAddFile")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeAddFile")
	}
	var err error
	*collection, _, err = unsafeGetCollection(collection.ID) // CollectionCacheLock is already locked
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	db := GetDB()
	// Persist the relational mapping; ignore duplicates.
	if err = db.Create(&CollectionFile{
		CollectionID: collection.ID,
		FileID:       fileDirectory,
	}).Error; err != nil {
		if !isDuplicateKeyError(err) {
			return err
		}
	}
	// Update RAM before recalculating size so the new file is reflected.
	if _, ok := CollectionFiles[collection.ID]; ok {
		CollectionFiles[collection.ID].Add(fileDirectory)
	}
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	if err = db.Model(&Collection{}).Where("id = ?", collection.ID).Update("size", collection.Size).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) AddFile(fileDirectory string) error {
	CollectionFilesMutex.Lock()
	CollectionFoldersMutex.RLock()
	FileCacheLock.RLock()
	CollectionCacheLock.Lock()
	defer CollectionFilesMutex.Unlock()
	defer CollectionFoldersMutex.RUnlock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeAddFile(fileDirectory)
}

func (collection *Collection) unsafeRemoveFile(fileDirectory string) error {
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFilesMutex before calling unsafeRemoveFile")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Read-Lock CollectionFoldersMutex before calling unsafeRemoveFile")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeRemoveFile")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeRemoveFile")
	}
	var err error
	*collection, _, err = unsafeGetCollection(collection.ID) // CollectionCacheLock is already locked
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	db := GetDB()
	if err := db.Where("collection_id = ? AND file_id = ?", collection.ID, fileDirectory).
		Delete(&CollectionFile{}).Error; err != nil {
		return err
	}
	// Update RAM before recalculating size so the removed file is reflected.
	if _, ok := CollectionFiles[collection.ID]; ok {
		CollectionFiles[collection.ID].Remove(fileDirectory)
	}
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	if err := db.Model(&Collection{}).Where("id = ?", collection.ID).Update("size", collection.Size).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) RemoveFile(fileDirectory string) error {
	CollectionFilesMutex.Lock()
	CollectionFoldersMutex.RLock()
	FileCacheLock.RLock()
	CollectionCacheLock.Lock()
	defer CollectionFilesMutex.Unlock()
	defer CollectionFoldersMutex.RUnlock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeRemoveFile(fileDirectory)
}
