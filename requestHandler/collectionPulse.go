package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
)

func PulseFileSubscribers(file database.FileData) {
	database.CollectionFilesMutex.RLock()
	var collectionIDs []string
	for collectionID, files := range database.CollectionFiles {
		if files.Contains(file.FileDirectory) {
			collectionIDs = append(collectionIDs, collectionID)
		}
	}
	database.CollectionFilesMutex.RUnlock()

	ActiveWebsocketsMutex.RLock()
	var connections []globals.WebsocketInfo
	for conn, connData := range ActiveWebsockets {
		for _, collectionID := range collectionIDs {
			if connData.SubscribedCollections[collectionID] {
				connections = append(connections, globals.WebsocketInfo{Conn: conn, Data: &connData})
				break
			}
		}
	}
	ActiveWebsocketsMutex.RUnlock()

	for _, connection := range connections {
		go func(connection globals.WebsocketInfo) {
			connection.Data.Mutex.Lock()
			defer connection.Data.Mutex.Unlock()
			if err := connection.Conn.WriteJSON(map[string]interface{}{
				"type": "file_update",
				"data": FileUpdate{Toggle: true, Replace: true, File: file},
			}); err != nil {
				ActiveWebsocketsMutex.Lock()
				delete(ActiveWebsockets, connection.Conn)
				ActiveWebsocketsMutex.Unlock()
				connection.Conn.Close()
			}
		}(connection)
	}
}

func PulseCollectionSubscribers(collection database.Collection) {
	if collection.ID == "" {
		return
	}
	var connectionsToUpdate []globals.WebsocketInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if _, ok := connData.SubscribedCollections[collection.ID]; ok {
			connectionsToUpdate = append(connectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		var token string
		ci.Data.Mutex.Lock()
		if ci.Data.UserInfo.Email != "" || ci.Data.UserInfo.HashedPassword != "" {
			if !accounts.AuthenticateHashed(ci.Data.UserInfo.Email, ci.Data.UserInfo.HashedPassword) {
				ci.Data.Mutex.Unlock()
				continue
			}
			accountInfo, _ := database.FindUserByEmail(ci.Data.UserInfo.Email)
			token = accountInfo.Token
		} else {
			token = ci.Data.UserInfo.Token
		}
		err := ci.Conn.WriteJSON(map[string]interface{}{
			"type": "get_collection_response",
			"data": Collection(collection).getCollectionResponse(token),
		})
		if err != nil {
			ci.Data.Mutex.Unlock()
			ActiveWebsocketsMutex.Lock()
			delete(ActiveWebsockets, ci.Conn)
			ActiveWebsocketsMutex.Unlock()
			ci.Conn.Close()
			continue
		}
		err = ci.Conn.WriteJSON(map[string]interface{}{
			"type": "collection_card_update",
			"data": CollectionCardData{
				CollectionID:   collection.ID,
				CollectionName: collection.Name,
				Size:           int64(collection.Size),
				FileCount:      len(collection.GetFiles()),
				FolderCount:    len(collection.GetCollections()),
				EditorCount:    len(collection.GetEditors()),
				Timestamp:      collection.Timestamp,
			},
		})
		ci.Data.Mutex.Unlock()
		if err != nil {
			ActiveWebsocketsMutex.Lock()
			delete(ActiveWebsockets, ci.Conn)
			ActiveWebsocketsMutex.Unlock()
			ci.Conn.Close()
		}
	}
}
