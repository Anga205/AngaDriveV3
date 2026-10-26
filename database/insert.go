package database

import (
	"angadrive/storage"
	"math/rand"
	"strings"
	"time"
)

func (collection *Collection) Insert() error {
	if collection.ID == "" {
		collection.ID = generateNewCollectionID()
	}
	if collection.Timestamp == 0 {
		collection.Timestamp = time.Now().Unix()
	}
	db := GetDB()
	if err := db.Create(&collection).Error; err != nil {
		return err
	}
	// All cache updates are synchronous so callers can rely on the collection
	// being fully present in RAM immediately after Insert returns. This also
	// avoids races between the insertion and subsequent mutations.
	CollectionCacheLock.Lock()
	CollectionCache[collection.ID] = *collection
	CollectionCacheLock.Unlock()

	CollectionFilesMutex.Lock()
	if _, ok := CollectionFiles[collection.ID]; !ok {
		CollectionFiles[collection.ID] = NewFileSet()
	}
	CollectionFilesMutex.Unlock()

	CollectionFoldersMutex.Lock()
	if _, ok := CollectionFolders[collection.ID]; !ok {
		CollectionFolders[collection.ID] = NewCollectionSet()
	}
	CollectionFoldersMutex.Unlock()

	UserCollectionsMutex.Lock()
	for _, editor := range collection.GetEditors() {
		if _, ok := UserCollections[editor]; ok {
			UserCollections[editor].Add(collection.ID)
		}
	}
	UserCollectionsMutex.Unlock()
	return nil
}

func (user Account) Insert() error {
	db := GetDB()
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	UserAccountsByEmailMutex.Lock()
	UserAccountsByEmail[user.Email] = user
	UserAccountsByEmailMutex.Unlock()

	UserAccountsByTokenMutex.Lock()
	UserAccountsByToken[user.Token] = user
	UserAccountsByTokenMutex.Unlock()
	return nil
}

func PushTimeStamp(timestamp int64) {
	go func() {
		TimeStampsMutex.Lock()
		defer TimeStampsMutex.Unlock()
		TimeStamps = append(TimeStamps, timestamp)
	}()
	GetDB().Model(&Activity{}).Create(&Activity{Timestamps: timestamp})
}

// GetTimeStamps returns a copy of the in-RAM activity timestamps. Callers must
// use this instead of reading the TimeStamps global directly so the read is
// synchronized with PushTimeStamp.
func GetTimeStamps() []int64 {
	TimeStampsMutex.RLock()
	defer TimeStampsMutex.RUnlock()
	out := make([]int64, len(TimeStamps))
	copy(out, TimeStamps)
	return out
}

func isFileNamePresent(filename string) bool {
	file, _ := GetFile(filename)
	return file.FileDirectory == filename
}

// GenerateUniqueFileName returns a random filename (preserving the original
// extension) that does not collide with an existing file.
func GenerateUniqueFileName(filename string) string {
	return storage.UniqueFileName(filename, isFileNamePresent)
}

func (file FileData) Insert() error {
	db := GetDB()
	if err := db.Create(&file).Error; err != nil {
		return err
	}

	// Synchronous cache updates (see Collection.Insert for rationale).
	UserFilesMutex.Lock()
	if _, ok := UserFiles[file.AccountToken]; ok {
		UserFiles[file.AccountToken].Add(file.FileDirectory)
	}
	UserFilesMutex.Unlock()

	FileCacheLock.Lock()
	FileCache[file.FileDirectory] = file
	FileCacheLock.Unlock()

	return nil
}

func generateRandomCollectionID() string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	allowed_characters := "1234567890qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	var sb strings.Builder
	for range [15]int{} {
		randomIndex := random.Intn(len(allowed_characters))
		sb.WriteString(string(allowed_characters[randomIndex]))
	}
	return sb.String()
}

func generateNewCollectionID() string {
	var collectionID string
	for {
		collectionID = generateRandomCollectionID()
		fetchCollection, _ := GetCollection(collectionID)
		if fetchCollection.ID == "" {
			break
		}
	}
	return collectionID
}

func InsertNewCollection(collection Collection) (Collection, error) {

	if err := collection.Insert(); err != nil {
		return Collection{}, err
	}

	return collection, nil

}
