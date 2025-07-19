package database

import (
	"fmt"
	"strings"
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
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeAddFolder")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		return fmt.Errorf("please Read-Lock FileCacheLock before calling unsafeAddFolder")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFoldersMutex before calling unsafeAddFolder")
	}
	// CollectionCacheLock is already locked
	// we're doing this to ensure that the collection still exists and is up-to-date by the time the locks were acquired and the function was called
	*collection, _, err = unsafeGetCollection(collection.ID)
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	folders := collection.GetCollections()
	folders = append(folders, folder)
	collection.Collections = strings.Join(folders, ",")
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err = db.Save(collection).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	if _, ok := CollectionFolders[collection.ID]; ok {
		CollectionFolders[collection.ID].Add(folder)
	}
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) AddFolder(folder string) error {
	CollectionCacheLock.Lock()
	FileCacheLock.RLock()
	CollectionFoldersMutex.Lock()
	defer CollectionFoldersMutex.Unlock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeAddFolder(folder)
}

func (collection *Collection) unsafeRemoveFolder(folder string) error {
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeRemoveFolder")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFoldersMutex before calling unsafeRemoveFolder")
	}
	var err error
	*collection, _, err = unsafeGetCollection(collection.ID) // CollectionCacheLock is already locked
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	folders := strings.Split(collection.Collections, ",")
	for i, f := range folders {
		if strings.TrimSpace(f) == folder {
			folders = append(folders[:i], folders[i+1:]...)
			break
		}
	}
	collection.Collections = strings.Join(folders, ",")
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	if _, ok := CollectionFolders[collection.ID]; ok {
		CollectionFolders[collection.ID].Remove(folder)
	}
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) RemoveFolder(folder string) error {
	CollectionFoldersMutex.Lock()
	CollectionCacheLock.Lock()
	FileCacheLock.RLock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	defer CollectionFoldersMutex.Unlock()
	return collection.unsafeRemoveFolder(folder)
}

func unsafeCalculateCollectionSize(collection Collection, excludeCollections map[string]bool, excludeFiles map[string]bool) int {
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		panic("please Read-Lock FileCacheLock before calling unsafeCalculateCollectionSize")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		panic("please Lock CollectionCacheLock before calling unsafeCalculateCollectionSize")
	}
	excludeCollections[collection.ID] = true
	files := collection.GetFiles()
	output := 0
	for _, file := range files {
		fileData, _, err := unsafeGetFile(file) // Rlock FileCacheLock before calling this function
		if err != nil {
			continue
		}
		if excludeFiles[fileData.FileDirectory] {
			continue
		}
		output += int(fileData.FileSize)
		excludeFiles[fileData.FileDirectory] = true
	}
	collections := collection.GetCollections()
	for _, subCollectionID := range collections {
		if excludeCollections[subCollectionID] {
			continue
		}
		collection, _, err := unsafeGetCollection(subCollectionID) // lock CollectionCacheLock before calling this function
		if err != nil {
			continue
		}
		output += unsafeCalculateCollectionSize(collection, excludeCollections, excludeFiles)
	}
	return output
}

func unsafeUpdateParentCollectionSizes(collectionID string, collectionsAlreadyUpdated *map[string]bool) {
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		panic("Please Lock CollectionCacheLock before calling unsafeUpdateParentCollectionSizes")
	}
	db := GetDB()
	var collections []Collection
	db.Where("collections LIKE ?", "%"+collectionID+"%").Find(&collections)
	for _, collection := range collections {
		if (*collectionsAlreadyUpdated)[collection.ID] {
			continue
		}
		(*collectionsAlreadyUpdated)[collection.ID] = true
		collection.Size = unsafeCalculateCollectionSize(collection, make(map[string]bool), make(map[string]bool))
		if err := db.Save(&collection).Error; err != nil {
			fmt.Printf("Error updating collection size for %s: %v\n", collection.ID, err)
			continue
		}
		CollectionCache[collection.ID] = collection // lock CollectionCacheLock before calling this function
		unsafeUpdateParentCollectionSizes(collection.ID, collectionsAlreadyUpdated)
	}
}

func (collection *Collection) unsafeAddFile(fileDirectory string) error {
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeAddFile")
	}
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFilesMutex before calling unsafeAddFile")
	}
	var err error
	*collection, _, err = unsafeGetCollection(collection.ID) // CollectionCacheLock is already locked
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	files := strings.Split(collection.Files, ",")
	for _, f := range files {
		if strings.TrimSpace(f) == fileDirectory {
			return nil
		}
	}
	files = append(files, fileDirectory)
	collection.Files = strings.Join(files, ",")
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	if _, ok := CollectionFiles[collection.ID]; ok {
		CollectionFiles[collection.ID].Add(fileDirectory)
	}
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) AddFile(fileDirectory string) error {
	CollectionCacheLock.Lock()
	CollectionFilesMutex.Lock()
	FileCacheLock.RLock()
	defer FileCacheLock.RUnlock()
	defer CollectionFilesMutex.Unlock()
	defer CollectionCacheLock.Unlock()
	return collection.unsafeAddFile(fileDirectory)
}

func (collection *Collection) unsafeRemoveFile(fileDirectory string) error {
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		return fmt.Errorf("please Lock CollectionCacheLock before calling unsafeRemoveFile")
	}
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		return fmt.Errorf("please Lock CollectionFilesMutex before calling unsafeRemoveFile")
	}
	var err error
	*collection, _, err = unsafeGetCollection(collection.ID) // CollectionCacheLock is already locked
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", collection.ID, err)
	}
	files := strings.Split(collection.Files, ",")
	for i, f := range files {
		if strings.TrimSpace(f) == fileDirectory {
			files = append(files[:i], files[i+1:]...)
			break
		}
	}
	collection.Files = strings.Join(files, ",")
	collection.Size = unsafeCalculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	CollectionCache[collection.ID] = *collection // CollectionCacheLock is already locked
	if _, ok := CollectionFiles[collection.ID]; ok {
		CollectionFiles[collection.ID].Remove(fileDirectory)
	}
	unsafeUpdateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	return nil
}

func (collection *Collection) RemoveFile(fileDirectory string) error {
	CollectionFilesMutex.Lock()
	CollectionCacheLock.Lock()
	FileCacheLock.RLock()
	defer FileCacheLock.RUnlock()
	defer CollectionCacheLock.Unlock()
	defer CollectionFilesMutex.Unlock()
	return collection.unsafeRemoveFile(fileDirectory)
}
