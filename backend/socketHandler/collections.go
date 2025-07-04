package socketHandler

import (
	"fmt"
	"service/accounts"
	"service/database"

	"github.com/gorilla/websocket"
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
		Dependant:   "",
	}
	collection, err := database.InsertNewCollection(collection)
	fmt.Println("New collection created:", collection.Name)
	go CollectionPulse(true, collection)
	if err != nil {
		return fmt.Errorf("failed to create collection: %v", err)
	}
	// TODO: Add new collection Pulser
	return nil
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
	fmt.Println("Collection pulse triggered for collection:", collection.Name)
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if !(connData.UserInfo.Email == "" || connData.UserInfo.HashedPassword == "") && (connData.UserInfo.Token == "") {
			connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	fmt.Println("Found", len(connectionsToUpdate), "websocket connections to update for collection pulse")
	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData *WebsocketData, newCollection database.Collection, update CollectionCardUpdate) {
			fmt.Println("Sending collection update to websocket")
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if connData.UserInfo.Email != "" || connData.UserInfo.HashedPassword != "" {
				if !accounts.AuthenticateHashed(connData.UserInfo.Email, connData.UserInfo.HashedPassword) {
					fmt.Println("Authentication failed for websocket connection, skipping collection update")
					return
				}
				accountInfo, _ := database.FindUserByEmail(connData.UserInfo.Email)
				if !newCollection.IsEditor(accountInfo.Token) {
					fmt.Println("User is not an editor of the collection, skipping collection update")
					return
				}
			} else {
				if !newCollection.IsEditor(connData.UserInfo.Token) {
					fmt.Println("User is not an editor of the collection, skipping collection update")
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
