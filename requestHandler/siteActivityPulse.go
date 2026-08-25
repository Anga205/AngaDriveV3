package requestHandler

import (
	"angadrive/database"
	"angadrive/globals"
	"angadrive/info"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func SiteActivityPulse() {
	if !database.IsInitialized() {
		return
	}
	database.PushTimeStamp(time.Now().Unix())
	x_axis, y_axis := info.GetLastXDaysCounts()
	graphData := GraphData{
		XAxis:       x_axis,
		YAxis:       y_axis,
		Label:       "Site Activity",
		BeginAtZero: true,
	}

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
		}(ci.Conn, ci.Data)
	}
}
