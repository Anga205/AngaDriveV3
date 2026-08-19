package database

import (
	"sync"
)

var (
	wg = &sync.WaitGroup{}
)

func loadUserFiles() {
	UserFilesMutex.Lock()
	defer UserFilesMutex.Unlock()
	defer wg.Done()

	var accounts []Account
	err := GetDB().Find(&accounts).Error
	if err != nil {
		panic("failed to load accounts: " + err.Error())
	}

	for _, account := range accounts {
		var files []FileData
		err := GetDB().Where("account_token = ?", account.Token).Find(&files).Error
		if err != nil {
			panic("failed to load files for account token " + account.Token + ": " + err.Error())
		}
		if UserFiles[account.Token] == nil {
			UserFiles[account.Token] = NewFileSet()
		}
		UserFiles[account.Token].Set(files)
	}
}

func loadUserCollections() {
	UserCollectionsMutex.Lock()
	defer UserCollectionsMutex.Unlock()
	defer wg.Done()

	var accounts []Account
	err := GetDB().Find(&accounts).Error
	if err != nil {
		panic("failed to load accounts for collections: " + err.Error())
	}

	for _, account := range accounts {
		var collections []Collection
		err := GetDB().Where("editors LIKE ?", "%"+account.Token+"%").Find(&collections).Error
		if err != nil {
			panic("failed to load collections for account token " + account.Token + ": " + err.Error())
		}
		if UserCollections[account.Token] == nil {
			UserCollections[account.Token] = NewCollectionSet()
		}
		UserCollections[account.Token].Set(collections)
	}
}

func loadCollectionFiles() {
	CollectionFilesMutex.Lock()
	defer CollectionFilesMutex.Unlock()
	defer wg.Done()

	var mappings []CollectionFile
	err := GetDB().Find(&mappings).Error
	if err != nil {
		panic("failed to load collection_files: " + err.Error())
	}
	for _, m := range mappings {
		if CollectionFiles[m.CollectionID] == nil {
			CollectionFiles[m.CollectionID] = NewFileSet()
		}
		CollectionFiles[m.CollectionID].Add(m.FileID)
	}
}

func loadCollectionFolders() {
	CollectionFoldersMutex.Lock()
	defer CollectionFoldersMutex.Unlock()
	defer wg.Done()

	var mappings []CollectionChild
	err := GetDB().Find(&mappings).Error
	if err != nil {
		panic("failed to load collection_children: " + err.Error())
	}
	for _, m := range mappings {
		if CollectionFolders[m.ParentCollectionID] == nil {
			CollectionFolders[m.ParentCollectionID] = NewCollectionSet()
		}
		CollectionFolders[m.ParentCollectionID].Add(m.ChildCollectionID)
	}
}

func loadTimeStamps() {
	TimeStampsMutex.Lock()
	defer TimeStampsMutex.Unlock()

	var timestamps []int64
	err := GetDB().Model(&Activity{}).Pluck("timestamps", &timestamps).Error
	if err != nil {
		panic("failed to load timestamps: " + err.Error())
	}

	TimeStamps = timestamps
}

func loadUserAccountsByEmail() {
	UserAccountsByEmailMutex.Lock()
	defer UserAccountsByEmailMutex.Unlock()
	defer wg.Done()

	var accounts []Account
	err := GetDB().Find(&accounts).Error
	if err != nil {
		panic("failed to load accounts by email: " + err.Error())
	}

	for _, account := range accounts {
		UserAccountsByEmail[account.Email] = account
	}
}

func loadUserAccountsByToken() {
	UserAccountsByTokenMutex.Lock()
	defer UserAccountsByTokenMutex.Unlock()
	defer wg.Done()

	var accounts []Account
	err := GetDB().Find(&accounts).Error
	if err != nil {
		panic("failed to load accounts by token: " + err.Error())
	}

	for _, account := range accounts {
		UserAccountsByToken[account.Token] = account
	}
}

func loadFiles() {
	FileCacheLock.Lock()
	defer FileCacheLock.Unlock()
	defer wg.Done()

	var files []FileData
	err := GetDB().Find(&files).Error
	if err != nil {
		panic("failed to load files: " + err.Error())
	}

	for _, file := range files {
		FileCache[file.FileDirectory] = file
	}
}

func loadCollections() {
	CollectionCacheLock.Lock()
	defer CollectionCacheLock.Unlock()
	defer wg.Done()

	var collections []Collection
	err := GetDB().Find(&collections).Error
	if err != nil {
		panic("failed to load collections: " + err.Error())
	}
	for _, collection := range collections {
		CollectionCache[collection.ID] = collection
	}
}

func LoadCache() {
	wg.Add(8)
	go loadUserFiles()
	go loadUserCollections()
	go loadCollectionFiles()
	go loadCollectionFolders()
	go loadFiles()
	go loadCollections()
	go loadUserAccountsByEmail()
	go loadUserAccountsByToken()
	wg.Wait()
}
