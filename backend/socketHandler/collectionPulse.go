package socketHandler

import (
	"service/accounts"
	"service/database"
)

func PulseCollectionSubscribers(collection database.Collection) {
	if collection.ID == "" {
		return
	}
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if _, ok := connData.SubscribedCollections[collection.ID]; ok {
			connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		var token string
		ci.data.Mutex.Lock()
		if ci.data.UserInfo.Email != "" || ci.data.UserInfo.HashedPassword != "" {
			if !accounts.AuthenticateHashed(ci.data.UserInfo.Email, ci.data.UserInfo.HashedPassword) {
				ci.data.Mutex.Unlock()
				continue
			}
			accountInfo, _ := database.FindUserByEmail(ci.data.UserInfo.Email)
			token = accountInfo.Token
		} else {
			token = ci.data.UserInfo.Token
		}
		ci.conn.WriteJSON(map[string]interface{}{
			"type": "get_collection_response",
			"data": Collection(collection).getCollectionResponse(token),
		})
		ci.data.Mutex.Unlock()
	}
}
