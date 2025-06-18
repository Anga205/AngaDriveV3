package database

import (
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

func GetFile(file_directory string) (FileData, error) {
	FileCacheLock.RLock()
	data, found := FileCache[file_directory]
	FileCacheLock.RUnlock()

	if found {
		return data, nil
	}

	db := GetDB()
	var fileData FileData
	err := db.Where("file_directory = ?", file_directory).First(&fileData).Error
	if err != nil {
		return FileData{}, err
	}

	FileCacheLock.Lock()
	FileCache[file_directory] = fileData
	FileCacheLock.Unlock()

	return fileData, nil
}
