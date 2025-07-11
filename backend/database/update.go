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
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	go func() {
		CollectionCacheLock.Lock()
		defer CollectionCacheLock.Unlock()
		CollectionCache[collection.ID] = *collection
	}()
	go func() {
		CollectionFoldersMutex.Lock()
		defer CollectionFoldersMutex.Unlock()
		if _, ok := CollectionFolders[collection.ID]; ok {
			CollectionFolders[collection.ID].Add(folder)
		}
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
	db := GetDB()
	if err := db.Save(collection).Error; err != nil {
		return err
	}
	go func() {
		CollectionCacheLock.Lock()
		defer CollectionCacheLock.Unlock()
		CollectionCache[collection.ID] = *collection
	}()
	go func() {
		CollectionFoldersMutex.Lock()
		defer CollectionFoldersMutex.Unlock()
		if _, ok := CollectionFolders[collection.ID]; ok {
			CollectionFolders[collection.ID].Remove(folder)
		}
	}()
	return nil
}
