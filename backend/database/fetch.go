package database

import (
	"gorm.io/gorm"
)

func FindUserByEmail(email string) (Account, error) {
	UserAccountsMutex.RLock()
	user, found := UserAccounts[email]
	UserAccountsMutex.RUnlock()
	if found {
		return user, nil
	}

	db := GetDB()
	var dbUser Account
	db = db.Session(&gorm.Session{
		// you have to manually set the logger to silent
		Logger: db.Logger.LogMode(0),
		// for some reason, gorm will print an error if it can't find row
		// instead of just return the error
	})
	err := db.Where("email = ?", email).First(&dbUser).Error
	if err == nil {
		go func() {
			UserAccountsMutex.Lock()
			UserAccounts[email] = dbUser
			UserAccountsMutex.Unlock()
		}()
	}
	return dbUser, err
}
