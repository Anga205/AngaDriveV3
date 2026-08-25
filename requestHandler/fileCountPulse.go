package requestHandler

import (
	"angadrive/database"
	"angadrive/globals"

	"github.com/gorilla/websocket"
)

var fileCount int64

func initFileCount() {
	fileCount, _ = database.CountFiles()
}

func FileCountPulse() {
	initFileCount()
	var connectionsToUpdate []globals.WebsocketInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if connData.HomePageUpdates {
			connectionsToUpdate = append(connectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData *globals.WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if !connData.HomePageUpdates {
				return
			}
			err := conn.WriteJSON(map[string]any{
				"type": "files_hosted_count",
				"data": fileCount,
			})
			if err != nil {
				ActiveWebsocketsMutex.Lock()
				delete(ActiveWebsockets, conn)
				ActiveWebsocketsMutex.Unlock()
				conn.Close()
			}
		}(ci.Conn, ci.Data)
	}
}
