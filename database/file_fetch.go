package database

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

func GetFile(fileDirectory string) (FileData, error) {
	FileCacheLock.RLock()
	fileData, inCache, err := unsafeGetFile(fileDirectory)
	FileCacheLock.RUnlock()
	if err != nil {
		return FileData{}, err
	}
	if !inCache {
		go func() {
			FileCacheLock.Lock()
			defer FileCacheLock.Unlock()
			fileData, _, err := unsafeGetFile(fileDirectory)
			if err != nil {
				return
			}
			FileCache[fileDirectory] = fileData
		}()
	}
	return fileData, nil
}

func CheckForFilesWithSha256sum(sha256sum string) bool {
	// if the file is in the cache or database, return true
	FileCacheLock.RLock()
	for _, file := range FileCache {
		if file.Sha256sum == sha256sum {
			FileCacheLock.RUnlock()
			return true
		}
	}
	FileCacheLock.RUnlock()
	db := GetDB()
	var count int64
	err := db.Model(&FileData{}).Where("sha256sum = ?", sha256sum).Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}
