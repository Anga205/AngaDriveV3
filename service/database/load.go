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

	var accountTokens []string
	err := GetDB().Model(&FileData{}).Distinct().Pluck("account_token", &accountTokens).Error
	if err != nil {
		panic("failed to load account tokens: " + err.Error())
	}

	for _, token := range accountTokens {
		var files []FileData
		err := GetDB().Where("account_token = ?", token).Find(&files).Error
		if err != nil {
			panic("failed to load files for account token " + token + ": " + err.Error())
		}
		UserFiles[token] = files
	}
	fmt.Println("User files loaded successfully.")
}

func loadUserCollections() {
	fmt.Println("Loading user collections...")
	UserCollectionsMutex.Lock()
	defer UserCollectionsMutex.Unlock()
	defer wg.Done()

	var accountTokens []string
	err := GetDB().Model(&Collection{}).Distinct().Pluck("editors", &accountTokens).Error
	if err != nil {
		panic("failed to load account tokens for collections: " + err.Error())
	}

	for _, token := range accountTokens {
		var collections []Collection
		err := GetDB().Where("editors = ?", token).Find(&collections).Error
		if err != nil {
			panic("failed to load collections for account token " + token + ": " + err.Error())
		}
		UserCollections[token] = collections
	}
	fmt.Println("User collections loaded successfully.")
}

func loadCollectionFiles() {
	fmt.Println("Loading collection files...")
	CollectionFilesMutex.Lock()
	defer CollectionFilesMutex.Unlock()
	defer wg.Done()

	var collectionIDs []string
	err := GetDB().Model(&FileData{}).Distinct().Pluck("file_directory", &collectionIDs).Error
	if err != nil {
		panic("failed to load collection IDs: " + err.Error())
	}

	for _, id := range collectionIDs {
		var files []FileData
		err := GetDB().Where("file_directory = ?", id).Find(&files).Error
		if err != nil {
			panic("failed to load files for collection ID " + id + ": " + err.Error())
		}
		CollectionFiles[id] = files
	}
	fmt.Println("Collection files loaded successfully.")
}

func loadCollectionFolders() {
	fmt.Println("Loading collection folders...")
	CollectionFoldersMutex.Lock()
	defer CollectionFoldersMutex.Unlock()
	defer wg.Done()

	var collectionIDs []string
	err := GetDB().Model(&Collection{}).Distinct().Pluck("id", &collectionIDs).Error
	if err != nil {
		panic("failed to load collection IDs for folders: " + err.Error())
	}

	for _, id := range collectionIDs {
		var collections []Collection
		err := GetDB().Where("id = ?", id).Find(&collections).Error
		if err != nil {
			panic("failed to load folders for collection ID " + id + ": " + err.Error())
		}
		CollectionFolders[id] = collections
	}
	fmt.Println("Collection folders loaded successfully.")
}

func loadFiles() {
	fmt.Println("Loading files...")
	FilesMutex.Lock()
	defer FilesMutex.Unlock()
	defer wg.Done()

	var files []FileData
	err := GetDB().Find(&files).Error
	if err != nil {
		panic("failed to load files: " + err.Error())
	}

	for _, file := range files {
		Files[file.FileDirectory] = file
	}
	fmt.Println("Files loaded successfully.")
}

func LoadCache() {
	wg.Add(5)

	go loadUserFiles()
	go loadUserCollections()
	go loadCollectionFiles()
	go loadCollectionFolders()
	go loadFiles()

	wg.Wait()
}
