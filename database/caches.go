package database

import (
	"sync"
)

type FileSet struct {
	mu   sync.RWMutex
	data map[string]struct{}
}

func NewFileSet() *FileSet {
	return &FileSet{
		data: make(map[string]struct{}),
	}
}

func (s *FileSet) Add(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[value] = struct{}{}
}

func (s *FileSet) Remove(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, value)
}

func (s *FileSet) Contains(value string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[value]
	return exists
}

func (s *FileSet) Array() []FileData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]FileData, 0, len(s.data))
	for key := range s.data {
		file, err := GetFile(key)
		if err != nil {
			continue // Skip if file not found
		}
		keys = append(keys, file)
	}
	return keys
}

func (s *FileSet) Set(values []FileData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.data {
		delete(s.data, key)
	}
	for _, value := range values {
		s.data[value.FileDirectory] = struct{}{}
	}
}

// Keys returns the raw file directory keys without hitting the database.
func (s *FileSet) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

type CollectionSet struct {
	mu   sync.RWMutex
	data map[string]struct{}
}

func NewCollectionSet() *CollectionSet {
	return &CollectionSet{
		data: make(map[string]struct{}),
	}
}

func (s *CollectionSet) Add(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[value] = struct{}{}
}

func (s *CollectionSet) Remove(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, value)
}

func (s *CollectionSet) Contains(value string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[value]
	return exists
}

func (s *CollectionSet) Array() []Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]Collection, 0, len(s.data))
	for key := range s.data {
		collection, err := GetCollection(key)
		if err != nil {
			continue // Skip if collection not found
		}
		keys = append(keys, collection)
	}
	return keys
}

func (s *CollectionSet) Set(values []Collection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.data {
		delete(s.data, key)
	}
	for _, value := range values {
		s.data[value.ID] = struct{}{}
	}
}

// Keys returns the raw collection ID keys without hitting the database.
func (s *CollectionSet) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

var (
	UserFiles      = make(map[string]*FileSet)
	UserFilesMutex sync.RWMutex

	UserCollections      = make(map[string]*CollectionSet)
	UserCollectionsMutex sync.RWMutex

	CollectionFiles      = make(map[string]*FileSet)
	CollectionFilesMutex sync.RWMutex

	CollectionFolders      = make(map[string]*CollectionSet)
	CollectionFoldersMutex sync.RWMutex

	TimeStamps      = []int64{}
	TimeStampsMutex sync.RWMutex

	UserAccountsByEmail      = make(map[string]Account)
	UserAccountsByEmailMutex sync.RWMutex

	UserAccountsByToken      = make(map[string]Account)
	UserAccountsByTokenMutex sync.RWMutex

	FileCache     = make(map[string]FileData)
	FileCacheLock = sync.RWMutex{}

	CollectionCache     = make(map[string]Collection)
	CollectionCacheLock = sync.RWMutex{}
)
