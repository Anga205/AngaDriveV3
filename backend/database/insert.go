package database

import (
	"math/rand"
	"strings"
	"time"
)

func PushTimeStamp(timestamp int64) {
	TimeStampsMutex.Lock()
	TimeStamps = append(TimeStamps, timestamp)
	TimeStampsMutex.Unlock()

	GetDB().Model(&Activity{}).Create(&Activity{Timestamps: timestamp})
}

func InsertNewUser(user Account) error {
	UserAccountsByEmailMutex.Lock()
	UserAccountsByEmail[user.Email] = user
	UserAccountsByEmailMutex.Unlock()

	UserAccountsByTokenMutex.Lock()
	UserAccountsByToken[user.Token] = user
	UserAccountsByTokenMutex.Unlock()

	db := GetDB()
	return db.Create(&user).Error
}

const allowed_values = "qwertyuiopasdfghjklzxcvbnm1234567890"

func GenerateUniqueFileName(filename string) string {
	var generated_name string
	for {
		generated_name = generateFileName(filename)
		if !IsFileNamePresent(generated_name) {
			break
		}
	}
	return generated_name
}

func generateFileName(filename string) string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	for range [12]int{} {
		randomIndex := random.Intn(len(allowed_values))
		sb.WriteString(string(allowed_values[randomIndex]))
	}
	generated_name := sb.String()

	if len(strings.Split(filename, ".")) > 1 {
		generated_name = generated_name + "." + strings.Split(filename, ".")[len(strings.Split(filename, "."))-1]
	}
	return generated_name
}

func IsFileNamePresent(filename string) bool {
	// Check in database
	db := GetDB()
	var fileData FileData
	result := db.Where("filename = ?", filename).First(&fileData)
	if result.Error == nil {
		return true // Filename exists in the database
	}

	// Check in UserFiles cache
	UserFilesMutex.Lock()
	defer UserFilesMutex.Unlock()
	for _, fileSet := range UserFiles {
		for file := range fileSet.data {
			if file == filename {
				return true // Filename exists in the cache
			}
		}
	}

	return false
}

func InsertNewFile(fileData FileData) error {
	db := GetDB()
	if err := db.Create(&fileData).Error; err != nil {
		return err
	}

	go func() {
		UserFilesMutex.Lock()
		defer UserFilesMutex.Unlock()

		if _, ok := UserFiles[fileData.AccountToken]; ok {
			UserFiles[fileData.AccountToken].Add(fileData.FileDirectory)
		}
	}()

	go func() {
		FileCacheLock.Lock()
		defer FileCacheLock.Unlock()
		FileCache[fileData.FileDirectory] = fileData
	}()

	return nil
}
