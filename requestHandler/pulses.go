package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
	"angadrive/info"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func PulseFileSubscribers(file database.FileData) {
	database.CollectionFilesMutex.RLock()
	var collectionIDs []string
	for collectionID, files := range database.CollectionFiles {
		if files.Contains(file.FileDirectory) {
			collectionIDs = append(collectionIDs, collectionID)
		}
	}
	database.CollectionFilesMutex.RUnlock()

	ActiveWebsocketsMutex.RLock()
	var connections []globals.WebsocketInfo
	for conn, connData := range ActiveWebsockets {
		for _, collectionID := range collectionIDs {
			if connData.SubscribedCollections[collectionID] {
				connections = append(connections, globals.WebsocketInfo{Conn: conn, Data: &connData})
				break
			}
		}
	}
	ActiveWebsocketsMutex.RUnlock()

	for _, connection := range connections {
		go func(connection globals.WebsocketInfo) {
			connection.Data.Mutex.Lock()
			defer connection.Data.Mutex.Unlock()
			if err := connection.Conn.WriteJSON(map[string]interface{}{
				"type": "file_update",
				"data": FileUpdate{Toggle: true, Replace: true, File: file},
			}); err != nil {
				ActiveWebsocketsMutex.Lock()
				delete(ActiveWebsockets, connection.Conn)
				ActiveWebsocketsMutex.Unlock()
				connection.Conn.Close()
			}
		}(connection)
	}
}

func PulseCollectionSubscribers(collection database.Collection) {
	if collection.ID == "" {
		return
	}
	var connectionsToUpdate []globals.WebsocketInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if _, ok := connData.SubscribedCollections[collection.ID]; ok {
			connectionsToUpdate = append(connectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		var token string
		ci.Data.Mutex.Lock()
		if ci.Data.UserInfo.Email != "" || ci.Data.UserInfo.HashedPassword != "" {
			if !accounts.AuthenticateHashed(ci.Data.UserInfo.Email, ci.Data.UserInfo.HashedPassword) {
				ci.Data.Mutex.Unlock()
				continue
			}
			accountInfo, _ := database.FindUserByEmail(ci.Data.UserInfo.Email)
			token = accountInfo.Token
		} else {
			token = ci.Data.UserInfo.Token
		}
		err := ci.Conn.WriteJSON(map[string]interface{}{
			"type": "get_collection_response",
			"data": Collection(collection).getCollectionResponse(token),
		})
		if err != nil {
			ci.Data.Mutex.Unlock()
			ActiveWebsocketsMutex.Lock()
			delete(ActiveWebsockets, ci.Conn)
			ActiveWebsocketsMutex.Unlock()
			ci.Conn.Close()
			continue
		}
		err = ci.Conn.WriteJSON(map[string]interface{}{
			"type": "collection_card_update",
			"data": CollectionCardData{
				CollectionID:   collection.ID,
				CollectionName: collection.Name,
				Size:           int64(collection.Size),
				FileCount:      len(collection.GetFiles()),
				FolderCount:    len(collection.GetCollections()),
				EditorCount:    len(collection.GetEditors()),
				Timestamp:      collection.Timestamp,
			},
		})
		ci.Data.Mutex.Unlock()
		if err != nil {
			ActiveWebsocketsMutex.Lock()
			delete(ActiveWebsockets, ci.Conn)
			ActiveWebsocketsMutex.Unlock()
			ci.Conn.Close()
		}
	}
}

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
		}(ci.Conn, ci.Data)
	}
}

func UserFilesPulse(file FileUpdate) {
	go FileCountPulse()
	var connectionsToUpdate []globals.WebsocketInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		if (connData.UserInfo.Email != "" && connData.UserInfo.HashedPassword != "") || (connData.UserInfo.Token != "") {
			connectionsToUpdate = append(connectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connData})
		}
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(conn *websocket.Conn, connData *globals.WebsocketData) {
			connData.Mutex.Lock()
			defer connData.Mutex.Unlock()
			if connData.UserInfo.Email != "" || connData.UserInfo.HashedPassword != "" {
				if !accounts.AuthenticateHashed(connData.UserInfo.Email, connData.UserInfo.HashedPassword) {
					return
				}
				accountInfo, _ := database.FindUserByEmail(connData.UserInfo.Email)
				if accountInfo.Token != file.File.AccountToken {
					return
				}
				err := conn.WriteJSON(map[string]interface{}{
					"type": "file_update",
					"data": file,
				})
				if err != nil {
					ActiveWebsocketsMutex.Lock()
					delete(ActiveWebsockets, conn)
					ActiveWebsocketsMutex.Unlock()
					conn.Close()
					return
				}
			} else {
				if connData.UserInfo.Token != file.File.AccountToken {
					return
				}
				err := conn.WriteJSON(map[string]interface{}{
					"type": "file_update",
					"data": file,
				})
				if err != nil {
					ActiveWebsocketsMutex.Lock()
					delete(ActiveWebsockets, conn)
					ActiveWebsocketsMutex.Unlock()
					conn.Close()
					return
				}
			}
		}(ci.Conn, ci.Data)
	}
}

func SiteActivityPulse() {
	if !database.IsInitialized() {
		return
	}
	database.PushTimeStamp(time.Now().Unix())
	xAxis, yAxis := info.GetLastXDaysCounts()
	graphData := GraphData{
		XAxis:       xAxis,
		YAxis:       yAxis,
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

const DaysTracked = info.DaysTracked

var (
	LastXDays    [DaysTracked]string
	SpaceUsedArr [DaysTracked]int64
)

func SpaceUsedPulse() {
	newLastXDays, newSpaceUsedArr, err := info.GetSpaceUsedGraph()
	if err != nil {
		return
	}
	if LastXDays == newLastXDays && SpaceUsedArr == newSpaceUsedArr {
		return
	}
	LastXDays = newLastXDays
	SpaceUsedArr = newSpaceUsedArr
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
