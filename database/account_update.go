package database

import "fmt"

func (oldInfo Account) Update(newInfo Account) error { // this assumes token can not change
	if oldInfo.Token != newInfo.Token {
		return fmt.Errorf("token cannot be changed, old: %s, new: %s", oldInfo.Token, newInfo.Token)
	}
	db := GetDB()
	err := db.Model(&Account{}).Where("token = ?", oldInfo.Token).Updates(newInfo).Error
	if err != nil {
		return err
	}
	// Cache updates are synchronous so callers can rely on the new values being
	// visible immediately after Update returns (e.g. a password upgrade followed
	// by an authentication check).
	UserAccountsByEmailMutex.Lock()
	if oldInfo.Email != newInfo.Email {
		delete(UserAccountsByEmail, oldInfo.Email)
	}
	UserAccountsByEmail[newInfo.Email] = newInfo
	UserAccountsByEmailMutex.Unlock()

	UserAccountsByTokenMutex.Lock()
	UserAccountsByToken[newInfo.Token] = newInfo
	UserAccountsByTokenMutex.Unlock()
	return nil
}
