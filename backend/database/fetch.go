package database

import (
	"strings"

	"gorm.io/gorm"
)

func FindUserByEmail(email string) (Account, error) {
	UserAccountsByEmailMutex.RLock()
	user, found := UserAccountsByEmail[email]
	UserAccountsByEmailMutex.RUnlock()
	if found {
		return user, nil
	}

	db := GetDB()
	var dbUser Account
	db = db.Session(&gorm.Session{
		// you have to manually set the logger to silent
		Logger: db.Logger.LogMode(0),
		// for some reason, gorm will print an error if it can't find row
		// instead of just return the error
	})
	err := db.Where("email = ?", email).First(&dbUser).Error
	if err == nil {
		go func() {
			UserAccountsByEmailMutex.Lock()
			UserAccountsByEmail[email] = dbUser
			UserAccountsByEmailMutex.Unlock()
		}()
	}
	return dbUser, err
}

func FindUserByToken(token string) (Account, error) {
	UserAccountsByTokenMutex.RLock()
	user, found := UserAccountsByToken[token]
	UserAccountsByTokenMutex.RUnlock()
	if found {
		return user, nil
	}

	db := GetDB()
	var dbUser Account
	db = db.Session(&gorm.Session{
		// you have to manually set the logger to silent
		Logger: db.Logger.LogMode(0),
		// for some reason, gorm will print an error if it can't find row
		// instead of just return the error
	})
	err := db.Where("token = ?", token).First(&dbUser).Error
	if err == nil {
		go func() {
			UserAccountsByTokenMutex.Lock()
			UserAccountsByToken[token] = dbUser
			UserAccountsByTokenMutex.Unlock()
		}()
	}
	return dbUser, err
}

func GetUserFiles(token string) ([]FileData, error) {
	UserFilesMutex.RLock()
	files, found := UserFiles[token]
	if found {
		UserFilesMutex.RUnlock()
		return files.Array(), nil
	}
	UserFilesMutex.RUnlock()
	db := GetDB()
	var dbFiles []FileData
	err := db.Where("account_token = ?", token).Find(&dbFiles).Error
	if err != nil {
		return nil, err
	}
	go func() {
		UserFilesMutex.Lock()
		defer UserFilesMutex.Unlock()
		if _, ok := UserFiles[token]; !ok {
			UserFiles[token] = NewFileSet()
		}
		UserFiles[token].Set(dbFiles)
	}()
	return dbFiles, nil
}

func unsafeGetFile(fileDirectory string) (FileData, bool, error) {
	if fileData, ok := FileCache[fileDirectory]; ok {
		return fileData, true, nil
	}
	db := GetDB()
	var fileData FileData
	err := db.Where("file_directory = ?", fileDirectory).First(&fileData).Error
	if err != nil {
		return FileData{}, false, err
	}
	return fileData, false, nil
}

func GetFile(file_directory string) (FileData, error) {
	FileCacheLock.RLock()
	fileData, inCache, err := unsafeGetFile(file_directory)
	FileCacheLock.RUnlock()
	if err != nil {
		return FileData{}, err
	}
	if !inCache {
		go func() {
			FileCacheLock.Lock()
			defer FileCacheLock.Unlock()
			fileData, _, err := unsafeGetFile(file_directory)
			if err != nil {
				return
			}
			FileCache[file_directory] = fileData
		}()
	}
	return fileData, nil
}

func unsafeGetCollection(collectionID string) (Collection, bool, error) {
	if collection, ok := CollectionCache[collectionID]; ok {
		return collection, true, nil
	}
	db := GetDB()
	var collection Collection
	err := db.Where("id = ?", collectionID).First(&collection).Error
	if err != nil {
		return Collection{}, false, err
	}
	return collection, false, nil
}

func GetCollection(collectionID string) (Collection, error) {
	CollectionCacheLock.RLock()
	data, inCache, err := unsafeGetCollection(collectionID)
	CollectionCacheLock.RUnlock()

	if err != nil {
		return Collection{}, err
	}
	if !inCache {
		go func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			data, _, err := unsafeGetCollection(collectionID)
			if err != nil {
				return
			}
			CollectionCache[collectionID] = data
		}()
	}
	return data, nil
}

func CheckForFilesWithMd5sum(md5sum string) bool {
	// if the file is in the cache or database, return true
	FileCacheLock.RLock()
	for _, file := range FileCache {
		if file.Md5sum == md5sum {
			FileCacheLock.RUnlock()
			return true
		}
	}
	FileCacheLock.RUnlock()
	db := GetDB()
	var count int64
	err := db.Model(&FileData{}).Where("md5sum = ?", md5sum).Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}

func GetCumulativeUserCount() (int64, error) {
	db := GetDB()
	var count int64
	err := db.Model(&Account{}).Distinct("token").Count(&count).Error
	if err != nil {
		return 0, err
	}

	var fileCount int64
	err = db.Model(&FileData{}).
		Distinct("account_token").
		Where("account_token NOT IN (SELECT token FROM accounts)").
		Count(&fileCount).Error
	if err != nil {
		return 0, err
	}

	var collectionCount int64
	err = db.Model(&Collection{}). // TODO: this is technically wrong since editors is a comma-separated string that for now just so happens to only contain one value.
					Distinct("editors").
					Where("editors NOT IN (SELECT token FROM accounts)").
					Where("editors NOT IN (SELECT account_token FROM file_data)").
					Count(&collectionCount).Error
	if err != nil {
		return 0, err
	}
	return count + fileCount + collectionCount, nil
}

func CountFiles() (int64, error) {
	db := GetDB()
	var count int64
	err := db.Model(&FileData{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

type SizeAndTime struct {
	Size int64
	Time int64
}

func GetAllFileSizesAndTimes() ([]SizeAndTime, error) {
	db := GetDB()
	var files []FileData
	err := db.Find(&files).Error
	if err != nil {
		return nil, err
	}

	var fileSizesAndTimes []SizeAndTime
	for _, file := range files {
		fileSizesAndTimes = append(fileSizesAndTimes, SizeAndTime{
			Size: file.FileSize,
			Time: file.Timestamp,
		})
	}
	return fileSizesAndTimes, nil
}

func (s Collection) GetCollections() []string {
	if s.Collections == "" {
		return []string{}
	}
	collections := strings.Split(s.Collections, ",")
	for i := range collections {
		collections[i] = strings.TrimSpace(collections[i])
	}
	return collections
}

func (s Collection) GetFiles() []string {
	if s.Files == "" {
		return []string{}
	}
	files := strings.Split(s.Files, ",")
	for i := range files {
		files[i] = strings.TrimSpace(files[i])
	}
	return files
}

func (s Collection) GetEditors() []string {
	editors := strings.Split(s.Editors, ",")
	for i := range editors {
		editors[i] = strings.TrimSpace(editors[i])
	}
	return editors
}

func (s Collection) IsEditor(token string) bool {
	editors := s.GetEditors()
	for _, editor := range editors {
		if editor == token {
			return true
		}
	}
	return false
}

func (user Account) GetCollections() ([]Collection, error) {
	UserCollectionsMutex.RLock()
	collections, found := UserCollections[user.Token]
	UserCollectionsMutex.RUnlock()
	if found {
		return collections.Array(), nil
	}
	db := GetDB()
	var dbCollections []Collection
	err := db.Where("editors LIKE ?", "%"+user.Token+"%").Find(&dbCollections).Error
	if err != nil {
		return nil, err
	}
	go func() {
		UserCollectionsMutex.Lock()
		defer UserCollectionsMutex.Unlock()
		if _, ok := UserCollections[user.Token]; !ok {
			UserCollections[user.Token] = NewCollectionSet()
		}
		UserCollections[user.Token].Set(dbCollections)
	}()
	return dbCollections, nil
}
