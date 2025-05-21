package database

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
