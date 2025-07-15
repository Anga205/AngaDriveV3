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

func (collection *Collection) AddFolder(folder string) error {
	folders := collection.GetCollections()
	folders = append(folders, folder)
	collection.Collections = strings.Join(folders, ",")
	collection.Size = calculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	go func() {
		func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			CollectionCache[collection.ID] = *collection
		}()
		func() {
			CollectionFoldersMutex.Lock()
			defer CollectionFoldersMutex.Unlock()
			if _, ok := CollectionFolders[collection.ID]; ok {
				CollectionFolders[collection.ID].Add(folder)
			}
		}()
		updateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	}()
	return nil
}

func (collection *Collection) RemoveFolder(folder string) error {
	if !collection.hasFolder(folder) {
		return fmt.Errorf("folder %s not found in collection %s", folder, collection.ID)
	}
	folders := strings.Split(collection.Collections, ",")
	for i, f := range folders {
		if strings.TrimSpace(f) == folder {
			folders = append(folders[:i], folders[i+1:]...)
			break
		}
	}
	collection.Collections = strings.Join(folders, ",")
	collection.Size = calculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	go func() {
		func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			CollectionCache[collection.ID] = *collection
		}()
		func() {
			CollectionFoldersMutex.Lock()
			defer CollectionFoldersMutex.Unlock()
			if _, ok := CollectionFolders[collection.ID]; ok {
				CollectionFolders[collection.ID].Remove(folder)
			}
		}()
		updateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	}()
	return nil
}

func calculateCollectionSize(collection Collection, excludeCollections map[string]bool, excludeFiles map[string]bool) int {
	excludeCollections[collection.ID] = true
	files := collection.GetFiles()
	output := 0
	for _, file := range files {
		fileData, err := GetFile(file)
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
		collection, err := GetCollection(subCollectionID)
		if err != nil {
			continue
		}
		output += calculateCollectionSize(collection, excludeCollections, excludeFiles)
	}
	return output
}

func updateParentCollectionSizes(collectionID string, collectionsAlreadyUpdated *map[string]bool) {
	db := GetDB()
	var collections []Collection
	db.Where("collections LIKE ?", "%"+collectionID+"%").Find(&collections)
	for _, collection := range collections {
		if (*collectionsAlreadyUpdated)[collection.ID] {
			continue
		}
		(*collectionsAlreadyUpdated)[collection.ID] = true
		collection.Size = calculateCollectionSize(collection, make(map[string]bool), make(map[string]bool))
		if err := db.Save(&collection).Error; err != nil {
			fmt.Printf("Error updating collection size for %s: %v\n", collection.ID, err)
			continue
		}
		go func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			CollectionCache[collection.ID] = collection
		}()
		updateParentCollectionSizes(collection.ID, collectionsAlreadyUpdated)
	}
}

func (collection *Collection) AddFile(fileDirectory string) error {
	files := collection.GetFiles()
	files = append(files, fileDirectory)
	collection.Files = strings.Join(files, ",")
	collection.Size = calculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	go func() {
		func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			CollectionCache[collection.ID] = *collection
		}()
		func() {
			CollectionFilesMutex.Lock()
			defer CollectionFilesMutex.Unlock()
			if _, ok := CollectionFiles[collection.ID]; ok {
				CollectionFiles[collection.ID].Add(fileDirectory)
			}
		}()
		updateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	}()
	return nil
}

func (collection *Collection) RemoveFile(fileDirectory string) error {
	files := strings.Split(collection.Files, ",")
	for i, f := range files {
		if strings.TrimSpace(f) == fileDirectory {
			files = append(files[:i], files[i+1:]...)
			break
		}
	}
	collection.Files = strings.Join(files, ",")
	collection.Size = calculateCollectionSize(*collection, make(map[string]bool), make(map[string]bool))
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	go func() {
		func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			CollectionCache[collection.ID] = *collection
		}()
		func() {
			CollectionFilesMutex.Lock()
			defer CollectionFilesMutex.Unlock()
			if _, ok := CollectionFiles[collection.ID]; ok {
				CollectionFiles[collection.ID].Remove(fileDirectory)
			}
		}()
		updateParentCollectionSizes(collection.ID, &map[string]bool{collection.ID: true})
	}()
	return nil
}
