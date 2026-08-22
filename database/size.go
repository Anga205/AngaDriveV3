package database

import (
	"fmt"
)

// This file contains collection size computation and parent-size propagation.
//
// A collection's Size is the sum of the sizes of all files it contains
// directly, plus the sizes of all descendant collections (recursively). Files
// and collections are deduplicated across branches so a file/collection that
// appears in multiple branches is counted only once per computation.

// unsafeCalculateCollectionSize computes the recursive size of a collection.
//
// Locking: the caller MUST already hold, in order:
//
//	CollectionFilesMutex   (read or write)
//	CollectionFoldersMutex (read or write)
//	FileCacheLock          (read)
//	CollectionCacheLock    (write)
//
// The two "exclude" maps prevent double-counting and infinite recursion in
// cyclic collection graphs: once a collection/file has been visited it is
// skipped on subsequent visits. This is what makes cyclic graphs terminate.
func unsafeCalculateCollectionSize(collection Collection, excludeCollections map[string]bool, excludeFiles map[string]bool) int {
	if CollectionFilesMutex.TryLock() {
		defer CollectionFilesMutex.Unlock()
		panic("please Lock CollectionFilesMutex before calling unsafeCalculateCollectionSize")
	}
	if CollectionFoldersMutex.TryLock() {
		defer CollectionFoldersMutex.Unlock()
		panic("please Lock CollectionFoldersMutex before calling unsafeCalculateCollectionSize")
	}
	if FileCacheLock.TryLock() {
		defer FileCacheLock.Unlock()
		panic("please Read-Lock FileCacheLock before calling unsafeCalculateCollectionSize")
	}
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		panic("please Lock CollectionCacheLock before calling unsafeCalculateCollectionSize")
	}
	excludeCollections[collection.ID] = true
	files := unsafeGetCollectionFiles(collection.ID)
	output := 0
	for _, file := range files {
		fileData, _, err := unsafeGetFile(file) // RLock FileCacheLock before calling this function
		if err != nil {
			continue
		}
		if excludeFiles[fileData.FileDirectory] {
			continue
		}
		output += int(fileData.FileSize)
		excludeFiles[fileData.FileDirectory] = true
	}
	collections := unsafeGetCollectionFolders(collection.ID)
	for _, subCollectionID := range collections {
		if excludeCollections[subCollectionID] {
			continue
		}
		sub, _, err := unsafeGetCollection(subCollectionID) // lock CollectionCacheLock before calling this function
		if err != nil {
			continue
		}
		output += unsafeCalculateCollectionSize(sub, excludeCollections, excludeFiles)
	}
	return output
}

// unsafeUpdateParentCollectionSizes walks UP the collection graph from
// collectionID, recomputing the size of every ancestor collection and
// persisting the updated sizes.
//
// Locking: the caller MUST already hold CollectionCacheLock (write). It also
// requires CollectionFilesMutex/CollectionFoldersMutex (read) and FileCacheLock
// (read) because it calls unsafeCalculateCollectionSize.
//
// collectionsAlreadyUpdated prevents revisiting a collection that appears via
// multiple parent branches (and guards against cycles).
func unsafeUpdateParentCollectionSizes(collectionID string, collectionsAlreadyUpdated *map[string]bool) {
	if CollectionCacheLock.TryLock() {
		defer CollectionCacheLock.Unlock()
		panic("Please Lock CollectionCacheLock before calling unsafeUpdateParentCollectionSizes")
	}
	db := GetDB()
	var mappings []CollectionChild
	db.Where("child_collection_id = ?", collectionID).Find(&mappings)
	for _, m := range mappings {
		parentID := m.ParentCollectionID
		if (*collectionsAlreadyUpdated)[parentID] {
			continue
		}
		(*collectionsAlreadyUpdated)[parentID] = true
		parent, _, err := unsafeGetCollection(parentID)
		if err != nil {
			continue
		}
		parent.Size = unsafeCalculateCollectionSize(parent, make(map[string]bool), make(map[string]bool))
		if err := db.Model(&Collection{}).Where("id = ?", parentID).Update("size", parent.Size).Error; err != nil {
			fmt.Printf("Error updating collection size for %s: %v\n", parentID, err)
			continue
		}
		CollectionCache[parentID] = parent // lock CollectionCacheLock before calling this function
		unsafeUpdateParentCollectionSizes(parentID, collectionsAlreadyUpdated)
	}
}
