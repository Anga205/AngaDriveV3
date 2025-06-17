package socketHandler

import (
	"fmt"
	"service/info"
	"time"

	"github.com/gorilla/websocket"
)

func sysinfoPulse() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		sysInfo, _ := info.GetSysInfo()
		for conn, connMutex := range ActiveWebsockets {
			if !connMutex.HomePageUpdates {
				continue
			}
			go func(conn *websocket.Conn, connMutex WebsocketData) {
				connMutex.Mutex.Lock()
				defer connMutex.Mutex.Unlock()
				err := conn.WriteJSON(map[string]interface{}{
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
			}(conn, connMutex)
		}
	}
}
