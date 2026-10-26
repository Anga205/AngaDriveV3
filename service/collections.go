package service

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/protocol"
	"fmt"
)

// CreateNewCollection creates a collection owned by the authenticated user.
func CreateNewCollection(req protocol.CreateCollectionRequest) (string, error) {
	token, err := Authenticate(req.Auth)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %v", err)
	}
	collection := database.Collection{Name: req.CollectionName, Editors: token}
	collection, err = database.InsertNewCollection(collection)
	if err != nil {
		return "", fmt.Errorf("failed to create collection: %v", err)
	}
	go hub.PublishCollectionCard(true, collection)
	return collection.ID, nil
}

// DeleteCollection deletes a collection the user can edit.
func DeleteCollection(req protocol.DeleteCollectionRequest) (string, error) {
	token, err := Authenticate(req.Auth)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return "", fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return "", fmt.Errorf("user is not an editor of the collection")
	}
	if err := collection.Delete(); err != nil {
		return "", fmt.Errorf("failed to delete collection: %v", err)
	}
	go hub.PublishCollectionCard(false, collection)
	return collection.ID, nil
}

// GetUserCollections returns the collection cards for the authenticated user.
func GetUserCollections(req protocol.AuthInfo) ([]protocol.CollectionCardData, error) {
	token, err := Authenticate(req)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	collections, err := database.Account{Token: token}.GetCollections()
	if err != nil {
		return nil, fmt.Errorf("failed to get collections: %v", err)
	}
	cards := []protocol.CollectionCardData{}
	for _, collection := range collections {
		if collection.Dependant != "" {
			continue
		}
		cards = append(cards, protocol.CollectionCard(collection))
	}
	return cards, nil
}

// GetCollection returns the full payload for a collection.
func GetCollection(req protocol.GetCollectionRequest) (protocol.GetCollectionResponse, error) {
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	token, err := Authenticate(req.Auth)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	return protocol.BuildCollectionResponse(collection, token), nil
}

func updateCollectionFolders(collectionID, folderID string, auth protocol.AuthInfo, add bool) (protocol.GetCollectionResponse, error) {
	token, err := Authenticate(auth)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(collectionID)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return protocol.GetCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}
	if add {
		folder, err := database.GetCollection(folderID)
		if err != nil {
			return protocol.GetCollectionResponse{}, fmt.Errorf("failed to get folder: %v", err)
		}
		if err := collection.AddFolder(folder.ID); err != nil {
			return protocol.GetCollectionResponse{}, fmt.Errorf("failed to add folder: %v", err)
		}
	} else {
		if err := collection.RemoveFolder(folderID); err != nil {
			return protocol.GetCollectionResponse{}, fmt.Errorf("failed to remove folder: %v", err)
		}
	}
	go hub.PublishCollection(collection)
	return protocol.BuildCollectionResponse(collection, token), nil
}

// AddFolder adds an existing collection as a child folder.
func AddFolder(req protocol.AddFolderToCollectionRequest) (protocol.GetCollectionResponse, error) {
	return updateCollectionFolders(req.CollectionID, req.FolderID, req.Auth, true)
}

// RemoveFolder removes a child folder from a collection.
func RemoveFolder(req protocol.RemoveFolderFromCollectionRequest) (protocol.GetCollectionResponse, error) {
	return updateCollectionFolders(req.CollectionID, req.FolderID, req.Auth, false)
}

// CreateFolderInCollection creates a new child collection and adds it.
func CreateFolderInCollection(req protocol.CreateFolderInCollectionRequest) (protocol.GetCollectionResponse, error) {
	token, err := Authenticate(req.Auth)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return protocol.GetCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}
	folder, err := database.InsertNewCollection(database.Collection{Name: req.FolderName, Editors: token})
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("failed to create folder: %v", err)
	}
	if err := collection.AddFolder(folder.ID); err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("failed to add folder to collection: %v", err)
	}
	go hub.PublishCollectionCard(true, folder)
	go hub.PublishCollection(collection)
	return protocol.BuildCollectionResponse(collection, token), nil
}

func updateCollectionFiles(collectionID, fileDirectory string, auth protocol.AuthInfo, add bool) (protocol.GetCollectionResponse, error) {
	token, err := Authenticate(auth)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(collectionID)
	if err != nil {
		return protocol.GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return protocol.GetCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}
	if add {
		if err := collection.AddFile(fileDirectory); err != nil {
			return protocol.GetCollectionResponse{}, fmt.Errorf("failed to add file: %v", err)
		}
	} else {
		if err := collection.RemoveFile(fileDirectory); err != nil {
			return protocol.GetCollectionResponse{}, fmt.Errorf("failed to remove file: %v", err)
		}
	}
	go hub.PublishCollection(collection)
	return protocol.BuildCollectionResponse(collection, token), nil
}

// AddFileToCollection adds a file to a collection.
func AddFileToCollection(req protocol.AddFileToCollectionRequest) (protocol.GetCollectionResponse, error) {
	return updateCollectionFiles(req.CollectionID, req.FileDirectory, req.Auth, true)
}

// RemoveFileFromCollection removes a file from a collection.
func RemoveFileFromCollection(req protocol.RemoveFileFromCollectionRequest) (protocol.GetCollectionResponse, error) {
	return updateCollectionFiles(req.CollectionID, req.FileDirectory, req.Auth, false)
}

// BulkAddFilesToCollection adds many files to a collection (creating it if a
// name is supplied), collecting per-file errors.
func BulkAddFilesToCollection(req protocol.BulkAddFilesToCollectionRequest) (protocol.BulkAddFilesToCollectionResponse, error) {
	token, err := Authenticate(req.Auth)
	if err != nil {
		return protocol.BulkAddFilesToCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}

	collectionID := req.CollectionID
	createdNew := false
	if collectionID == "" && req.CollectionName != "" {
		collection, err := database.InsertNewCollection(database.Collection{Name: req.CollectionName, Editors: token})
		if err != nil {
			return protocol.BulkAddFilesToCollectionResponse{}, fmt.Errorf("failed to create collection: %v", err)
		}
		collectionID = collection.ID
		createdNew = true
	}
	if collectionID == "" {
		return protocol.BulkAddFilesToCollectionResponse{}, fmt.Errorf("collection id or collection name is required")
	}

	collection, err := database.GetCollection(collectionID)
	if err != nil {
		return protocol.BulkAddFilesToCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return protocol.BulkAddFilesToCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}

	added := []string{}
	errs := []protocol.FileDeleteError{}
	for _, fileDirectory := range req.FileDirectories {
		file, err := database.GetFile(fileDirectory)
		if err != nil {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: "file not found"})
			continue
		}
		if file.AccountToken != token {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: "unauthorized add attempt"})
			continue
		}
		if err := collection.AddFile(fileDirectory); err != nil {
			errs = append(errs, protocol.FileDeleteError{FileDirectory: fileDirectory, Error: err.Error()})
			continue
		}
		added = append(added, fileDirectory)
	}

	if createdNew {
		go hub.PublishCollectionCard(true, collection)
	}
	go hub.PublishCollection(collection)
	return protocol.BulkAddFilesToCollectionResponse{CollectionID: collectionID, Added: added, Errors: errs}, nil
}
