package service

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/hub"
	"angadrive/protocol"
	"fmt"
)

// RemoveAccount deletes a user account and all of its files and collections.
func RemoveAccount(req protocol.RemoveAccountRequest) (string, error) {
	files, collections, err := accounts.DeleteUser(accounts.DeleteUserRequest(req))
	if err != nil {
		return "", fmt.Errorf("failed to remove account: %v", err)
	}
	for _, file := range files {
		go func(file database.FileData) {
			affected, err := database.DeleteFile(file)
			if err == nil {
				for _, collection := range affected {
					hub.PublishCollection(collection)
				}
			}
			RemoveFileIfNoClonesExist(file)
		}(file)
	}
	for _, collection := range collections {
		go collection.Delete()
	}
	go hub.ForceLogout(req.Email)
	return "Account deleted successfully", nil
}
