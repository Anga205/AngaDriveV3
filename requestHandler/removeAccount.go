package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
	"fmt"

	"github.com/gorilla/websocket"
)

func logoutUser(email string) error {
	var collectionsToUpdate []globals.WebsocketInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if connData.UserInfo.Email == email && connData.UserInfo.HashedPassword != "" {
			collectionsToUpdate = append(collectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	ActiveWebsocketsMutex.Lock()
	for conn := range ActiveWebsockets {
		if ActiveWebsockets[conn].UserInfo.Email == email && ActiveWebsockets[conn].UserInfo.HashedPassword != "" {
			temp := ActiveWebsockets[conn]
			temp.UserInfo = globals.UserInfo{}
			ActiveWebsockets[conn] = temp
		}
	}
	ActiveWebsocketsMutex.Unlock()
	for _, ci := range collectionsToUpdate {
		go func(conn *websocket.Conn, connData *globals.WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			err := conn.WriteJSON(map[string]any{
				"type": "force_logout",
				"data": email,
			})
			if err != nil {
				ActiveWebsocketsMutex.Lock()
				delete(ActiveWebsockets, conn)
				ActiveWebsocketsMutex.Unlock()
				conn.Close()
			}
		}(ci.Conn, ci.Data)
	}
	return nil
}

func removeAccountHandler(req removeAccRequest) (string, error) {
	files, collections, err := accounts.DeleteUser(accounts.DeleteUserRequest(req))
	if err != nil {
		return "", fmt.Errorf("failed to remove account: %v", err)
	}
	for _, file := range files {
		go func(file database.FileData) {
			database.DeleteFile(file, PulseCollectionSubscribers)
			RemoveFileIfNoClonesExist(file)
		}(file)
	}
	for _, collection := range collections {
		go collection.Delete()
	}
	go logoutUser(req.Email)
	return "Account deleted successfully", nil
}
