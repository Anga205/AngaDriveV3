package socketHandler

import (
	"fmt"
	"service/accounts"
	"service/database"

	"github.com/gorilla/websocket"
)

func CreateNewCollection(req CreateCollectionRequest) (string, error) {
	userToken, err := req.Auth.GetToken()
	if err != nil {
		return "", fmt.Errorf("authentication failed: %v", err)
	}
	collection := database.Collection{
		Name:        req.CollectionName,
		Editors:     userToken,
		Files:       "",
		Collections: "",
		Size:        0,
		Dependant:   "",
	}
	collection, _ = database.InsertNewCollection(collection)
	go CollectionPulse(true, collection)
	return collection.ID, nil
}

func DeleteCollection(req DeleteCollectionRequest) (string, error) {
	userToken, err := req.Auth.GetToken()
	if err != nil {
		return "", fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return "", fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(userToken) {
		return "", fmt.Errorf("user is not an editor of the collection")
	}
	err = collection.Delete()
	if err != nil {
		return "", fmt.Errorf("failed to delete collection: %v", err)
	}
	go CollectionPulse(false, collection)
	return collection.ID, nil
}

func GetUserCollections(req AuthInfo) ([]CollectionCardData, error) {
	token, err := req.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	collections, err := database.Account{Token: token}.GetCollections()
	if err != nil {
		return nil, fmt.Errorf("failed to get collections: %v", err)
	}
	var collectionCards []CollectionCardData = []CollectionCardData{}
	for _, collection := range collections {
		if collection.Dependant != "" {
			continue
		} else {
			card := CollectionCardData{
				CollectionID:   collection.ID,
				CollectionName: collection.Name,
				Size:           int64(collection.Size),
				FileCount:      len(collection.GetFiles()),
				FolderCount:    len(collection.GetCollections()),
				EditorCount:    len(collection.GetEditors()),
				Timestamp:      collection.Timestamp,
			}
			collectionCards = append(collectionCards, card)
		}
	}
	return collectionCards, nil
}

func CollectionPulse(toggle bool, collection database.Collection) {
	collectionCard := CollectionCardUpdate{
		Toggle: toggle,
		Collection: CollectionCardData{
			CollectionID:   collection.ID,
			CollectionName: collection.Name,
			Size:           int64(collection.Size),
			FileCount:      len(collection.GetFiles()),
			FolderCount:    len(collection.GetCollections()),
			EditorCount:    len(collection.GetEditors()),
			Timestamp:      collection.Timestamp,
		},
	}
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if (connData.UserInfo.Email != "" && connData.UserInfo.HashedPassword != "") || (connData.UserInfo.Token != "") {
			connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData *WebsocketData, newCollection database.Collection, update CollectionCardUpdate) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if connData.UserInfo.Email != "" || connData.UserInfo.HashedPassword != "" {
				if !accounts.AuthenticateHashed(connData.UserInfo.Email, connData.UserInfo.HashedPassword) {
					return
				}
				accountInfo, _ := database.FindUserByEmail(connData.UserInfo.Email)
				if !newCollection.IsEditor(accountInfo.Token) {
					return
				}
			} else {
				if !newCollection.IsEditor(connData.UserInfo.Token) {
					return
				}
			}
			err := conn.WriteJSON(map[string]interface{}{
				"type": "collection_update",
				"data": update,
			})
			if err != nil {
				ActiveWebsocketsMutex.Lock()
				delete(ActiveWebsockets, conn)
				ActiveWebsocketsMutex.Unlock()
				conn.Close()
				return
			}
		}(ci.conn, &ci.data, collection, collectionCard)
	}
}

func GetCollection(req GetCollectionRequest) (GetCollectionResponse, error) {
	var resp GetCollectionResponse
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return resp, fmt.Errorf("failed to get collection: %v", err)
	}
	req.Auth.Token, err = req.Auth.GetToken()
	if err != nil {
		return resp, fmt.Errorf("authentication failed: %v", err)
	}
	return Collection(collection).getCollectionResponse(req.Auth.Token), nil
}

func updateCollectionFolders(collectionID, folderID string, auth AuthInfo, add bool) (GetCollectionResponse, error) {
	token, err := auth.GetToken()
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(collectionID)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return GetCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}
	if add {
		folder, err := database.GetCollection(folderID)
		if err != nil {
			return GetCollectionResponse{}, fmt.Errorf("failed to get folder: %v", err)
		}
		collection.AddFolder(folder.ID)
	} else {
		collection.RemoveFolder(folderID)
	}
	go PulseCollectionSubscribers(collection)
	return Collection(collection).getCollectionResponse(token), nil
}

func AddFolder(req AddFolderToCollectionRequest) (GetCollectionResponse, error) {
	return updateCollectionFolders(req.CollectionID, req.FolderID, req.Auth, true)
}

func RemoveFolder(req RemoveFolderFromCollectionRequest) (GetCollectionResponse, error) {
	return updateCollectionFolders(req.CollectionID, req.FolderID, req.Auth, false)
}

func CreateFolderInCollection(req CreateFolderInCollectionRequest) (GetCollectionResponse, error) {
	token, err := req.Auth.GetToken()
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return GetCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}
	folder := database.Collection{
		Name:        req.FolderName,
		Editors:     token,
		Collections: "",
		Files:       "",
		Size:        0,
	}
	folder, _ = database.InsertNewCollection(folder)
	err = collection.AddFolder(folder.ID)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("failed to add folder to collection: %v", err)
	}
	go CollectionPulse(true, folder)
	go PulseCollectionSubscribers(collection)
	return Collection(collection).getCollectionResponse(token), nil
}

func updateCollectionFiles(collectionID, fileDirectory string, auth AuthInfo, add bool) (GetCollectionResponse, error) {
	token, err := auth.GetToken()
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(collectionID)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(token) {
		return GetCollectionResponse{}, fmt.Errorf("user is not an editor of the collection")
	}
	if add {
		collection.AddFile(fileDirectory)
	} else {
		collection.RemoveFile(fileDirectory)
	}
	go PulseCollectionSubscribers(collection)
	return Collection(collection).getCollectionResponse(token), nil
}

func AddFileToCollection(req AddFileToCollectionRequest) (GetCollectionResponse, error) {
	return updateCollectionFiles(req.CollectionID, req.FileDirectory, req.Auth, true)
}

func RemoveFileFromCollection(req RemoveFileFromCollectionRequest) (GetCollectionResponse, error) {
	return updateCollectionFiles(req.CollectionID, req.FileDirectory, req.Auth, false)
}
