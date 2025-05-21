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
		if UserFiles[account] == nil {
			UserFiles[account] = NewFileSet()
		}
		UserFiles[account].Set(files)
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
		if UserCollections[account] == nil {
			UserCollections[account] = NewCollectionSet()
		}
		UserCollections[account].Set(collections)
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
		if CollectionFiles[collection] == nil {
			CollectionFiles[collection] = NewFileSet()
		}
		fileDirectoryList := collection.GetFiles()
		for _, file := range fileDirectoryList {
			var files []FileData
			err := GetDB().Where("file_directory = ?", file).Find(&files).Error
			if err != nil {
				panic("failed to load files for collection ID " + collection.ID + ": " + err.Error())
			}
			CollectionFiles[collection].Set(files)
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
		if CollectionFolders[collection] == nil {
			CollectionFolders[collection] = NewCollectionSet()
		}
		folderIDList := collection.GetCollections()
		for _, folderID := range folderIDList {
			var folders []Collection
			err := GetDB().Where("id = ?", folderID).Find(&folders).Error
			if err != nil {
				panic("failed to load folders for collection ID " + collection.ID + ": " + err.Error())
			}
			CollectionFolders[collection].Set(folders)
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

func LoadCache() {
	wg.Add(5)

	go loadUserFiles()
	go loadUserCollections()
	go loadCollectionFiles()
	go loadCollectionFolders()
	go loadTimeStamps()

	wg.Wait()
}
