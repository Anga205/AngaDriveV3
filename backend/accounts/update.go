package accounts

import (
	"fmt"
	"service/database"
)

func updateUserWithAuth(email, password string, updateFunc func(*database.Account)) (database.Account, error) {
	if !Authenticate(email, password) {
		return database.Account{}, fmt.Errorf("invalid credentials")
	}
	oldUserInfo, err := database.FindUserByEmail(email)
	if err != nil {
		return database.Account{}, err
	}
	newUserInfo := oldUserInfo
	updateFunc(&newUserInfo)
	err = oldUserInfo.Update(newUserInfo)
	if err != nil {
		return database.Account{}, err
	}
	return newUserInfo, nil
}

func ChangeUserEmail(request ChangeEmailRequest) (database.Account, error) {
	return updateUserWithAuth(request.OldEmail, request.Password, func(user *database.Account) {
		user.Email = request.NewEmail
	})
}

func ChangeUserPassword(request ChangePasswordRequest) (database.Account, error) {
	return updateUserWithAuth(request.Email, request.OldPassword, func(user *database.Account) {
		user.HashedPassword = request.NewPasswordHashed
	})
}

func ChangeUserDisplayName(request ChangeDisplayNameRequest) (database.Account, error) {
	return updateUserWithAuth(request.Email, request.Password, func(user *database.Account) {
		user.DisplayName = request.DisplayName
	})
}
