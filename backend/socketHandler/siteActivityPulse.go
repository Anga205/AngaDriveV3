package socketHandler

import (
	"fmt"
	"service/database"
	"service/info"
	"time"

	"github.com/gorilla/websocket"
)

func siteActivityPulse() {
	database.PushTimeStamp(time.Now().Unix())
	x_axis, y_axis := info.GetLast7DaysCounts()
	graphData := GraphData{
		XAxis:       x_axis,
		YAxis:       y_axis,
		Label:       "Site Activity",
		BeginAtZero: true,
	}
	ActiveWebsocketsMutex.RLock()
	defer ActiveWebsocketsMutex.RUnlock()
	for conn, connData := range ActiveWebsockets {
		if !connData.HomePageUpdates {
			continue
		}
		go func(conn *websocket.Conn, connData WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			err := conn.WriteJSON(map[string]interface{}{
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
		}(conn, connData)
	}
}
