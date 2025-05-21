package database

func UpdateUser(OldUserInfo Account, NewUserInfo Account) error {

	UserAccountsByEmailMutex.Lock()
	if OldUserInfo.Email != NewUserInfo.Email {
		delete(UserAccountsByEmail, OldUserInfo.Email)
		UserAccountsByEmail[NewUserInfo.Email] = NewUserInfo
	} else {
		UserAccountsByEmail[NewUserInfo.Email] = NewUserInfo
	}
	UserAccountsByEmailMutex.Unlock()

	UserAccountsByTokenMutex.Lock()
	UserAccountsByToken[OldUserInfo.Token] = NewUserInfo
	UserAccountsByTokenMutex.Unlock()

	db := GetDB()
	return db.Model(&Account{}).Where("email = ?", OldUserInfo.Email).Updates(NewUserInfo).Error
}
