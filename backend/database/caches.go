package database

import "sync"

var (
	UserFiles      = make(map[string][]FileData)
	UserFilesMutex sync.RWMutex

	UserCollections      = make(map[string][]Collection)
	UserCollectionsMutex sync.RWMutex

	CollectionFiles      = make(map[string][]FileData)
	CollectionFilesMutex sync.RWMutex

	CollectionFolders      = make(map[string][]Collection)
	CollectionFoldersMutex sync.RWMutex

	Files      = make(map[string]FileData)
	FilesMutex sync.RWMutex

	TimeStamps      = []int64{}
	TimeStampsMutex sync.RWMutex

	UserAccounts      = make(map[string]Account)
	UserAccountsMutex sync.RWMutex
)

func DeleteFileFromCaches(file FileData) {
	UserFilesMutex.Lock()
	for _, files := range UserFiles {
		for i, f := range files {
			if f == file {
				files = append(files[:i], files[i+1:]...)
			}
		}
	}
	UserFilesMutex.Unlock()

	CollectionFilesMutex.Lock()
	for _, collections := range CollectionFiles {
		for i, f := range collections {
			if f == file {
				collections = append(collections[:i], collections[i+1:]...)
			}
		}
	}
	CollectionFilesMutex.Unlock()
}

func DeleteCollectionFromCaches(collection Collection) {
	UserCollectionsMutex.Lock()
	CollectionFoldersMutex.Lock()
	for _, collections := range UserCollections {
		for i, c := range collections {
			if c == collection {
				collections = append(collections[:i], collections[i+1:]...)
			}
		}
	}
	defer UserCollectionsMutex.Unlock()

	for _, folders := range CollectionFolders {
		for i, c := range folders {
			if c == collection {
				folders = append(folders[:i], folders[i+1:]...)
			}
		}
	}
	defer CollectionFoldersMutex.Unlock()
}
