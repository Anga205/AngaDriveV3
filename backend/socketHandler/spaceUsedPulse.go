package socketHandler

import (
	"service/info"
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
		}(ci.conn, ci.data)
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
