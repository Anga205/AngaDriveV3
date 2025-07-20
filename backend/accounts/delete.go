package accounts

import (
	"fmt"
	"service/database"
)

func DeleteUser(RequestInfo DeleteUserRequest) ([]database.FileData, []database.Collection, error) {
	if !Authenticate(RequestInfo.Email, RequestInfo.Password) {
		return nil, nil, fmt.Errorf("invalid credentials")
	}
	user, _ := database.FindUserByEmail(RequestInfo.Email)
	files, _ := database.GetUserFiles(user.Token)
	collections, _ := user.GetCollections()
	err := user.Delete()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete user: %v", err)
	}
	return files, collections, nil
}
