package socketHandler

import (
	"fmt"
	"service/accounts"
	"service/database"
)

func CreateNewCollection(req CreateCollectionRequest) error {
	if req.Auth.Token == "" {
		if !accounts.Authenticate(req.Auth.Email, req.Auth.Password) {
			return fmt.Errorf("authentication failed")
		}
		user, _ := database.FindUserByEmail(req.Auth.Email)
		req.Auth.Token = user.Token
	}
	collection := database.Collection{
		Name:        req.CollectionName,
		Editors:     req.Auth.Token,
		Files:       "",
		Collections: "",
		Size:        0,
		Hidden:      false,
	}
	_, err := database.InsertNewCollection(collection)
	if err != nil {
		return fmt.Errorf("failed to create collection: %v", err)
	}
	// TODO: Add new collection Pulser
	return nil
}
