package socketHandler

import (
	"service/accounts"
	"service/database"

	"github.com/gorilla/websocket"
)

func UserFilesPulse(file FileUpdate) {
	go FileCountPulse()
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if !(connData.UserInfo.Email == "" || connData.UserInfo.HashedPassword == "") && (connData.UserInfo.Token == "") {
			connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData *WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if connData.UserInfo.Email != "" || connData.UserInfo.HashedPassword != "" {
				if !accounts.AuthenticateHashed(connData.UserInfo.Email, connData.UserInfo.HashedPassword) {
					return
				}
				accountInfo, _ := database.FindUserByEmail(connData.UserInfo.Email)
				if accountInfo.Token != file.File.AccountToken {
					return
				}
				err := conn.WriteJSON(map[string]interface{}{
					"type": "file_update",
					"data": file,
				})
				if err != nil {
					ActiveWebsocketsMutex.Lock()
					delete(ActiveWebsockets, conn)
					ActiveWebsocketsMutex.Unlock()
					conn.Close()
					return
				}
			} else {
				if connData.UserInfo.Token != file.File.AccountToken {
					return
				}
				err := conn.WriteJSON(map[string]interface{}{
					"type": "file_update",
					"data": file,
				})
				if err != nil {
					ActiveWebsocketsMutex.Lock()
					delete(ActiveWebsockets, conn)
					ActiveWebsocketsMutex.Unlock()
					conn.Close()
					return
				}
			}
		}(ci.conn, &ci.data)
	}
}
