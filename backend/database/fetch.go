package database

import (
	"gorm.io/gorm"
)

func FindUserByEmail(email string) (Account, error) {
	UserAccountsByEmailMutex.RLock()
	user, found := UserAccountsByEmail[email]
	UserAccountsByEmailMutex.RUnlock()
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
			UserAccountsByEmailMutex.Lock()
			UserAccountsByEmail[email] = dbUser
			UserAccountsByEmailMutex.Unlock()
		}()
	}
	return dbUser, err
}
