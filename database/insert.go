package database

import (
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
	go func() {
		CollectionCacheLock.Lock()
		defer CollectionCacheLock.Unlock()
		CollectionCache[collection.ID] = *collection
	}()
	go func() {
		editors := collection.GetEditors()
		for _, editor := range editors {
			if _, ok := UserCollections[editor]; ok {
				UserCollections[editor].Add(collection.ID)
			}
		}
	}()
	return nil
}

func (user Account) Insert() error {
	db := GetDB()
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	go func() {
		UserAccountsByEmailMutex.Lock()
		defer UserAccountsByEmailMutex.Unlock()
		UserAccountsByEmail[user.Email] = user
	}()

	go func() {
		UserAccountsByTokenMutex.Lock()
		defer UserAccountsByTokenMutex.Unlock()
		UserAccountsByToken[user.Token] = user
	}()
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

const allowed_filename_characters = "qwertyuiopasdfghjklzxcvbnm1234567890"

func isFileNamePresent(filename string) bool {

	file, _ := GetFile(filename)

	return file.FileDirectory == filename
}

func GenerateUniqueFileName(filename string) string {
	var generated_name string
	for {
		generated_name = generateFileName(filename)
		if !isFileNamePresent(generated_name) {
			break
		}
	}
	return generated_name
}

func generateFileName(filename string) string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	for range [12]int{} {
		randomIndex := random.Intn(len(allowed_filename_characters))
		sb.WriteString(string(allowed_filename_characters[randomIndex]))
	}
	generated_name := sb.String()

	if len(strings.Split(filename, ".")) > 1 {
		generated_name = generated_name + "." + strings.Split(filename, ".")[len(strings.Split(filename, "."))-1]
	}
	return generated_name
}

func (file FileData) Insert() error {
	db := GetDB()
	if err := db.Create(&file).Error; err != nil {
		return err
	}

	go func() {
		UserFilesMutex.Lock()
		defer UserFilesMutex.Unlock()

		if _, ok := UserFiles[file.AccountToken]; ok {
			UserFiles[file.AccountToken].Add(file.FileDirectory)
		}
	}()

	go func() {
		FileCacheLock.Lock()
		defer FileCacheLock.Unlock()
		FileCache[file.FileDirectory] = file
	}()

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
