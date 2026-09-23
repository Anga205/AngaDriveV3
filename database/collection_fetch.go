package database

import "strings"

func unsafeGetCollection(collectionID string) (Collection, bool, error) {
	if collection, ok := CollectionCache[collectionID]; ok {
		return collection, true, nil
	}
	db := GetDB()
	var collection Collection
	err := db.Where("id = ?", collectionID).First(&collection).Error
	if err != nil {
		return Collection{}, false, err
	}
	return collection, false, nil
}

func GetCollection(collectionID string) (Collection, error) {
	CollectionCacheLock.RLock()
	data, inCache, err := unsafeGetCollection(collectionID)
	CollectionCacheLock.RUnlock()

	if err != nil {
		return Collection{}, err
	}
	if !inCache {
		go func() {
			CollectionCacheLock.Lock()
			defer CollectionCacheLock.Unlock()
			data, _, err := unsafeGetCollection(collectionID)
			if err != nil {
				return
			}
			CollectionCache[collectionID] = data
		}()
	}
	return data, nil
}

func (s Collection) GetEditors() []string {
	editors := strings.Split(s.Editors, ",")
	for i := range editors {
		editors[i] = strings.TrimSpace(editors[i])
	}
	return editors
}

func (s Collection) IsEditor(token string) bool {
	editors := s.GetEditors()
	for _, editor := range editors {
		if editor == token {
			return true
		}
	}
	return false
}

func (user Account) GetCollections() ([]Collection, error) {
	UserCollectionsMutex.RLock()
	collections, found := UserCollections[user.Token]
	UserCollectionsMutex.RUnlock()
	if found {
		return collections.Array(), nil
	}
	db := GetDB()
	var dbCollections []Collection
	err := db.Where("editors LIKE ?", "%"+user.Token+"%").Find(&dbCollections).Error
	if err != nil {
		return nil, err
	}
	go func() {
		UserCollectionsMutex.Lock()
		defer UserCollectionsMutex.Unlock()
		if _, ok := UserCollections[user.Token]; !ok {
			UserCollections[user.Token] = NewCollectionSet()
		}
		UserCollections[user.Token].Set(dbCollections)
	}()
	return dbCollections, nil
}
