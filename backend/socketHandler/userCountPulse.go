package socketHandler

import (
	"fmt"
	"service/database"

	"github.com/gorilla/websocket"
)

var userCount int64

func initializeUserCount() {
	userCount, _ = database.GetCumulativeUserCount()
}

func UpdateUserCount() {
	localUserCount, err := database.GetCumulativeUserCount()
	if err != nil {
		fmt.Print("[socketHandler/userCountPulse.go] Error fetching user count: ", err)
	} else {
		if localUserCount != userCount {
			userCount = localUserCount
			go UserCountPulse()
		}
	}
}

func UserCountPulse() {
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if connData.HomePageUpdates {
			connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if !connData.HomePageUpdates {
				return
			}
			err := conn.WriteJSON(map[string]any{
				"type": "user_count",
				"data": userCount,
			})
			if err != nil {
				fmt.Printf("Error writing to websocket: %v\n", err)
				ActiveWebsocketsMutex.Lock()
				delete(ActiveWebsockets, conn)
				ActiveWebsocketsMutex.Unlock()
				conn.Close()
			}
		}(ci.conn, ci.data)
	}
}
