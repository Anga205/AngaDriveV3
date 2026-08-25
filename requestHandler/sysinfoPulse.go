package requestHandler

import (
	"angadrive/globals"
	"angadrive/info"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func sysinfoPulse() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var connectionsToUpdate []globals.WebsocketInfo
	for {
		<-ticker.C
		sysInfo, _ := info.GetSysInfo()

		connectionsToUpdate = connectionsToUpdate[:0] // Reset the slice for each tick

		ActiveWebsocketsMutex.RLock()
		for conn, connMutex := range ActiveWebsockets {
			if connMutex.HomePageUpdates {
				connectionsToUpdate = append(connectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connMutex})
			}
		}
		ActiveWebsocketsMutex.RUnlock()

		for _, c := range connectionsToUpdate {
			go func(conn *websocket.Conn, connMutex *globals.WebsocketData) {
				connMutex.Mutex.Lock()
				defer connMutex.Mutex.Unlock()
				err := conn.WriteJSON(map[string]any{
					"type": "system_information",
					"data": sysInfo,
				})
				if err != nil {
					fmt.Printf("Error writing to websocket: %v\n", err)
					ActiveWebsocketsMutex.Lock()
					delete(ActiveWebsockets, conn)
					ActiveWebsocketsMutex.Unlock()
					conn.Close()
				}
			}(c.Conn, c.Data)
		}
	}
}
