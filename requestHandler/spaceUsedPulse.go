package requestHandler

import (
	"angadrive/globals"
	"angadrive/info"
	"time"

	"github.com/gorilla/websocket"
)

const X = info.X

var (
	LastXDays    [X]string
	SpaceUsedArr [X]int64
)

func SpaceUsedPulse() {
	new_LastXDays, new_SpaceUsedArr, err := info.GetSpaceUsedGraph()
	if err != nil {
		return
	}
	if LastXDays == new_LastXDays && SpaceUsedArr == new_SpaceUsedArr {
		return
	}
	LastXDays = new_LastXDays
	SpaceUsedArr = new_SpaceUsedArr
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
				"type": "graph_data",
				"data": GraphData{
					XAxis:       LastXDays[:],
					YAxis:       SpaceUsedArr[:],
					Label:       "Space Used",
					BeginAtZero: false,
				},
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

func initSpaceUsedPulser() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		SpaceUsedPulse()
	}
}
