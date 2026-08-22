package database

// This file contains the collection → file and collection → collection
// relationship query/accessor layer.
//
// The relational tables (collection_files, collection_children) are the
// durable source of truth. At startup they are loaded into the RAM indexes
// CollectionFiles and CollectionFolders (see load.go). These accessors read
// from RAM first and fall back to a direct database query (populating RAM in
// the background) only if the index has not yet been built for a given ID.

// GetFiles returns the file-directory keys that belong to this collection.
//
// Fast path: reads CollectionFiles[ID] from RAM. If that index has not been
// populated yet (e.g. before LoadCache, or for a collection created by an
// external writer), it queries collection_files and populates the RAM set.
func (s Collection) GetFiles() []string {
	CollectionFilesMutex.RLock()
	set, found := CollectionFiles[s.ID]
	CollectionFilesMutex.RUnlock()
	if found {
		return set.Keys()
	}
	db := GetDB()
	var mappings []CollectionFile
	if err := db.Where("collection_id = ?", s.ID).Find(&mappings).Error; err != nil {
		return []string{}
	}
	keys := make([]string, 0, len(mappings))
	for _, m := range mappings {
		keys = append(keys, m.FileID)
	}
	go func() {
		CollectionFilesMutex.Lock()
		defer CollectionFilesMutex.Unlock()
		if _, ok := CollectionFiles[s.ID]; !ok {
			CollectionFiles[s.ID] = NewFileSet()
		}
		for _, k := range keys {
			CollectionFiles[s.ID].Add(k)
		}
	}()
	return keys
}

// GetCollections returns the child-collection IDs of this collection.
//
// Fast path: reads CollectionFolders[ID] from RAM, with the same relational
// fallback + lazy cache population as GetFiles.
func (s Collection) GetCollections() []string {
	CollectionFoldersMutex.RLock()
	set, found := CollectionFolders[s.ID]
	CollectionFoldersMutex.RUnlock()
	if found {
		return set.Keys()
	}
	db := GetDB()
	var mappings []CollectionChild
	if err := db.Where("parent_collection_id = ?", s.ID).Find(&mappings).Error; err != nil {
		return []string{}
	}
	keys := make([]string, 0, len(mappings))
	for _, m := range mappings {
		keys = append(keys, m.ChildCollectionID)
	}
	go func() {
		CollectionFoldersMutex.Lock()
		defer CollectionFoldersMutex.Unlock()
		if _, ok := CollectionFolders[s.ID]; !ok {
			CollectionFolders[s.ID] = NewCollectionSet()
		}
		for _, k := range keys {
			CollectionFolders[s.ID].Add(k)
		}
	}()
	return keys
}

// unsafeGetCollectionFiles reads a collection's file-directory keys directly
// from the RAM index without acquiring CollectionFilesMutex.
//
// The caller MUST already hold CollectionFilesMutex (read or write). This
// mirrors the unsafeGetCollection/unsafeGetFile convention: the name "unsafe"
// signals that no additional locking is performed.
func unsafeGetCollectionFiles(collectionID string) []string {
	set, ok := CollectionFiles[collectionID]
	if !ok {
		return []string{}
	}
	return set.Keys()
}

// unsafeGetCollectionFolders reads a collection's child-collection IDs directly
// from the RAM index without acquiring CollectionFoldersMutex.
//
// The caller MUST already hold CollectionFoldersMutex (read or write).
func unsafeGetCollectionFolders(collectionID string) []string {
	set, ok := CollectionFolders[collectionID]
	if !ok {
		return []string{}
	}
	return set.Keys()
}
