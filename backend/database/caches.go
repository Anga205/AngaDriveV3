package database

import "sync"

type FileSet struct {
	mu   sync.RWMutex
	data map[FileData]struct{}
}

func NewFileSet() *FileSet {
	return &FileSet{
		data: make(map[FileData]struct{}),
	}
}

func (s *FileSet) Add(value FileData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[value] = struct{}{}
}

func (s *FileSet) Remove(value FileData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, value)
}

func (s *FileSet) Contains(value FileData) bool {
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
		keys = append(keys, key)
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
		s.data[value] = struct{}{}
	}
}

func (s *FileSet) CommaSeparated() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result string
	for key := range s.data {
		result += key.FileDirectory + ","
	}
	if len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result
}

type CollectionSet struct {
	mu   sync.RWMutex
	data map[Collection]struct{}
}

func NewCollectionSet() *CollectionSet {
	return &CollectionSet{
		data: make(map[Collection]struct{}),
	}
}

func (s *CollectionSet) Add(value Collection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[value] = struct{}{}
}

func (s *CollectionSet) Remove(value Collection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, value)
}

func (s *CollectionSet) Contains(value Collection) bool {
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
		keys = append(keys, key)
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
		s.data[value] = struct{}{}
	}
}

var (
	UserFiles      = make(map[string]*FileSet)
	UserFilesMutex sync.RWMutex

	UserCollections      = make(map[string]*CollectionSet)
	UserCollectionsMutex sync.RWMutex

	CollectionFiles      = make(map[Collection]*FileSet)
	CollectionFilesMutex sync.RWMutex

	CollectionFolders      = make(map[Collection]*CollectionSet)
	CollectionFoldersMutex sync.RWMutex

	TimeStamps      = []int64{}
	TimeStampsMutex sync.RWMutex

	UserAccountsByEmail      = make(map[string]Account)
	UserAccountsByEmailMutex sync.RWMutex

	UserAccountsByToken      = make(map[string]Account)
	UserAccountsByTokenMutex sync.RWMutex
)
