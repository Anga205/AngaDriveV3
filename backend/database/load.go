package database

import (
	"fmt"
	"sync"
)

var (
	wg = &sync.WaitGroup{}
)

func loadUserFiles() {
	fmt.Println("Loading user files...")
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
	fmt.Println("User files loaded successfully.")
}

func loadUserCollections() {
	fmt.Println("Loading user collections...")
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
	fmt.Println("User collections loaded successfully.")
}

func loadCollectionFiles() {
	fmt.Println("Loading collection files...")
	CollectionFilesMutex.Lock()
	defer CollectionFilesMutex.Unlock()
	defer wg.Done()

	var collections []Collection
	err := GetDB().Find(&collections).Error
	if err != nil {
		panic("failed to load collections for files: " + err.Error())
	}
	for _, collection := range collections {
		if CollectionFiles[collection.ID] == nil {
			CollectionFiles[collection.ID] = NewFileSet()
		}
		fileDirectoryList := collection.GetFiles()
		for _, file := range fileDirectoryList {
			var files []FileData
			err := GetDB().Where("file_directory = ?", file).Find(&files).Error
			if err != nil {
				panic("failed to load files for collection ID " + collection.ID + ": " + err.Error())
			}
			CollectionFiles[collection.ID].Set(files)
		}
	}
	fmt.Println("Collection files loaded successfully.")
}

func loadCollectionFolders() {
	fmt.Println("Loading collection folders...")
	CollectionFoldersMutex.Lock()
	defer CollectionFoldersMutex.Unlock()
	defer wg.Done()

	var collections []Collection
	err := GetDB().Find(&collections).Error
	if err != nil {
		panic("failed to load collections for folders: " + err.Error())
	}
	for _, collection := range collections {
		if CollectionFolders[collection.ID] == nil {
			CollectionFolders[collection.ID] = NewCollectionSet()
		}
		folderIDList := collection.GetCollections()
		for _, folderID := range folderIDList {
			var folders []Collection
			err := GetDB().Where("id = ?", folderID).Find(&folders).Error
			if err != nil {
				panic("failed to load folders for collection ID " + collection.ID + ": " + err.Error())
			}
			CollectionFolders[collection.ID].Set(folders)
		}
	}
	fmt.Println("Collection folders loaded successfully.")
}

func loadTimeStamps() {
	fmt.Println("Loading timestamps...")
	TimeStampsMutex.Lock()
	defer TimeStampsMutex.Unlock()
	defer wg.Done()

	var timestamps []int64
	err := GetDB().Model(&Activity{}).Pluck("timestamps", &timestamps).Error
	if err != nil {
		panic("failed to load timestamps: " + err.Error())
	}

	TimeStamps = timestamps
	fmt.Println("Timestamps loaded successfully.")
}

func loadUserAccountsByEmail() {
	fmt.Println("Loading user accounts by email...")
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
	fmt.Println("User accounts by email loaded successfully.")
}

func loadUserAccountsByToken() {
	fmt.Println("Loading user accounts by token...")
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
	fmt.Println("User accounts by token loaded successfully.")
}

func loadFiles() {
	fmt.Println("Loading files...")
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
	fmt.Println("Files loaded successfully.")
}

func loadCollections() {
	fmt.Println("Loading collections...")
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
	fmt.Println("Collections loaded successfully.")
}

func LoadCache() {
	wg.Add(9)

	go loadUserFiles()
	go loadUserCollections()
	go loadCollectionFiles()
	go loadCollectionFolders()
	go loadTimeStamps()
	go loadFiles()
	go loadCollections()
	go loadUserAccountsByEmail()
	go loadUserAccountsByToken()

	wg.Wait()
}
