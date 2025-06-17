package socketHandler

import (
	"fmt"
	"service/accounts"
	"service/database"

	"github.com/gorilla/websocket"
)

func UserFilesPulse(file FileUpdate) {
	fmt.Println("UserFilesPulse called with file:", file)
	for conn, connData := range ActiveWebsockets {
		go func(conn *websocket.Conn, connData *WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if (connData.UserInfo.Email == "" || connData.UserInfo.HashedPassword == "") && (connData.UserInfo.Token == "") {
				return
			} else if connData.UserInfo.Email != "" || connData.UserInfo.HashedPassword != "" {
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
		}(conn, &connData)
	}
}
