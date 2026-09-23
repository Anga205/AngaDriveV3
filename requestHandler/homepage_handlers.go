package requestHandler

import (
	"angadrive/info"
	"encoding/json"

	"github.com/gorilla/websocket"
)

func handleEnableHomepageUpdates(conn *websocket.Conn, data json.RawMessage) {
	var enabled bool
	if err := json.Unmarshal(data, &enabled); err != nil {
		return
	}

	ActiveWebsocketsMutex.Lock()
	wsData := ActiveWebsockets[conn]
	wsData.HomePageUpdates = enabled
	ActiveWebsockets[conn] = wsData
	ActiveWebsocketsMutex.Unlock()

	xAxis, yAxis := info.GetLastXDaysCounts()
	siteActivityData := GraphData{
		XAxis:       xAxis,
		YAxis:       yAxis,
		Label:       "Site Activity",
		BeginAtZero: true,
	}
	sysinfo, _ := info.GetSysInfo()

	sendJSON(
		conn,
		map[string]interface{}{
			"type": "graph_data",
			"data": siteActivityData,
		})
	sendJSON(conn, map[string]interface{}{
		"type": "graph_data",
		"data": GraphData{XAxis: LastXDays[:], YAxis: SpaceUsedArr[:], Label: "Space Used", BeginAtZero: false},
	})
	sendJSON(conn, map[string]interface{}{
		"type": "user_count",
		"data": userCount,
	})
	sendJSON(conn, map[string]interface{}{
		"type": "files_hosted_count",
		"data": fileCount,
	})
	sendJSON(conn, map[string]interface{}{
		"type": "system_information",
		"data": sysinfo,
	})
}
