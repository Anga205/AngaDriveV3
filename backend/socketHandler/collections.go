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

type CollectionCardData struct {
	CollectionID   string `json:"id"`
	CollectionName string `json:"name"`
	Size           int64  `json:"size"`
	FileCount      int    `json:"file_count"`
	FolderCount    int    `json:"folder_count"`
	EditorCount    int    `json:"editor_count"`
}

func GetUserCollections(req AuthInfo) ([]CollectionCardData, error) {
	var user database.Account
	if req.Token == "" {
		if !accounts.Authenticate(req.Email, req.Password) {
			return nil, fmt.Errorf("authentication failed")
		}
		user, _ = database.FindUserByEmail(req.Email)
	} else {
		user = database.Account{
			Token: req.Token,
		}
	}
	collections, err := user.GetCollections()
	if err != nil {
		return nil, fmt.Errorf("failed to get collections: %v", err)
	}
	var collectionCards []CollectionCardData
	for _, collection := range collections {
		if collection.Hidden {
			continue
		} else {
			card := CollectionCardData{
				CollectionName: collection.Name,
				Size:           int64(collection.Size),
				FileCount:      len(collection.GetFiles()),
				FolderCount:    len(collection.GetCollections()),
				EditorCount:    len(collection.GetEditors()),
			}
			collectionCards = append(collectionCards, card)
		}
	}
	return collectionCards, nil
}
