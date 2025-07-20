package socketHandler

import (
	"fmt"
	"service/database"
	"service/info"
	"time"

	"github.com/gorilla/websocket"
)

func SiteActivityPulse() {
	database.PushTimeStamp(time.Now().Unix())
	x_axis, y_axis := info.GetLast7DaysCounts()
	graphData := GraphData{
		XAxis:       x_axis,
		YAxis:       y_axis,
		Label:       "Site Activity",
		BeginAtZero: true,
	}

	var connectionsToUpdate []connInfo

	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if connData.HomePageUpdates {
			connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: &connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()

	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData *WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			err := conn.WriteJSON(map[string]any{
				"type": "graph_data",
				"data": graphData,
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
