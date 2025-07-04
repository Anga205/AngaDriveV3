package socketHandler

import (
	"fmt"
	"service/accounts"
	"service/database"

	"github.com/gorilla/websocket"
)

func CreateNewCollection(req CreateCollectionRequest) error {
	userToken, err := req.Auth.GetToken()
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	collection := database.Collection{
		Name:        req.CollectionName,
		Editors:     userToken,
		Files:       "",
		Collections: "",
		Size:        0,
		Dependant:   "",
	}
	collection, err = database.InsertNewCollection(collection)
	go CollectionPulse(true, collection)
	if err != nil {
		return fmt.Errorf("failed to create collection: %v", err)
	}
	return nil
}

func DeleteCollection(req DeleteCollectionRequest) error {
	userToken, err := req.Auth.GetToken()
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	collection, err := database.GetCollection(req.CollectionID)
	if err != nil {
		return fmt.Errorf("failed to get collection: %v", err)
	}
	if !collection.IsEditor(userToken) {
		return fmt.Errorf("user is not an editor of the collection")
	}
	err = collection.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete collection: %v", err)
	}
	go CollectionPulse(false, collection)
	return nil
}

func GetUserCollections(req AuthInfo) ([]CollectionCardData, error) {
	var user database.Account
	token, err := req.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	user, _ = database.FindUserByToken(token)
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
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if !(connData.UserInfo.Email == "" || connData.UserInfo.HashedPassword == "") && (connData.UserInfo.Token == "") {
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
