package hub

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
	"angadrive/info"
	"angadrive/protocol"
	"fmt"
	"sync"
	"time"
)

// resolveToken returns the account token for a connection's session, or "" if
// the session is not authenticated.
func resolveToken(d globals.WebsocketData) string {
	if d.UserInfo.Email != "" && d.UserInfo.HashedPassword != "" {
		if !accounts.AuthenticateHashed(d.UserInfo.Email, d.UserInfo.HashedPassword) {
			return ""
		}
		account, err := database.FindUserByEmail(d.UserInfo.Email)
		if err != nil {
			return ""
		}
		return account.Token
	}
	return d.UserInfo.Token
}

// NotifyUser sends an arbitrary message to every connection authenticated as
// the given token.
func NotifyUser(token string, message map[string]interface{}) {
	broadcastEach(hasCredentials, func(d globals.WebsocketData) (interface{}, bool) {
		if resolveToken(d) != token {
			return nil, false
		}
		return message, true
	})
}

// PublishFileUpdate sends a file add/remove event to the owning user's
// connections and refreshes the hosted-file count for homepage subscribers.
func PublishFileUpdate(update protocol.FileUpdate) {
	go PublishFileCount()
	broadcastEach(hasCredentials, func(d globals.WebsocketData) (interface{}, bool) {
		if resolveToken(d) != update.File.AccountToken {
			return nil, false
		}
		return map[string]interface{}{"type": "file_update", "data": update}, true
	})
}

// PublishFileToSubscribers sends a file update to connections subscribed to any
// collection containing the file.
func PublishFileToSubscribers(file database.FileData) {
	database.CollectionFilesMutex.RLock()
	var collectionIDs []string
	for collectionID, files := range database.CollectionFiles {
		if files.Contains(file.FileDirectory) {
			collectionIDs = append(collectionIDs, collectionID)
		}
	}
	database.CollectionFilesMutex.RUnlock()

	subscribed := func(d globals.WebsocketData) bool {
		for _, id := range collectionIDs {
			if d.SubscribedCollections[id] {
				return true
			}
		}
		return false
	}
	broadcast(subscribed, map[string]interface{}{
		"type": "file_update",
		"data": protocol.FileUpdate{Toggle: true, Replace: true, File: file},
	})
}

// PublishCollection sends the full collection payload plus a card update to
// every connection subscribed to the collection.
func PublishCollection(collection database.Collection) {
	if collection.ID == "" {
		return
	}
	broadcastEach(func(d globals.WebsocketData) bool {
		return d.SubscribedCollections[collection.ID]
	}, func(d globals.WebsocketData) (interface{}, bool) {
		token := resolveToken(d)
		return map[string]interface{}{
			"type": "get_collection_response",
			"data": protocol.BuildCollectionResponse(collection, token),
		}, true
	})
	broadcastEach(func(d globals.WebsocketData) bool {
		return d.SubscribedCollections[collection.ID]
	}, func(d globals.WebsocketData) (interface{}, bool) {
		return map[string]interface{}{
			"type": "collection_card_update",
			"data": protocol.CollectionCard(collection),
		}, true
	})
}

// PublishCollectionCard sends a collection add/remove card update to every
// authenticated connection that is an editor of the collection.
func PublishCollectionCard(toggle bool, collection database.Collection) {
	update := protocol.CollectionCardUpdate{Toggle: toggle, Collection: protocol.CollectionCard(collection)}
	broadcastEach(hasCredentials, func(d globals.WebsocketData) (interface{}, bool) {
		if !collection.IsEditor(resolveToken(d)) {
			return nil, false
		}
		return map[string]interface{}{"type": "collection_update", "data": update}, true
	})
}

var (
	fileCount int64
	userCount int64
)

// InitCounters seeds the cached file/user counts from the database.
func InitCounters() {
	fileCount, _ = database.CountFiles()
	userCount, _ = database.GetCumulativeUserCount()
}

// FileCount returns the cached hosted-file count.
func FileCount() int64 { return fileCount }

// UserCount returns the cached user count.
func UserCount() int64 { return userCount }

// PublishFileCount refreshes and broadcasts the hosted-file count.
func PublishFileCount() {
	fileCount, _ = database.CountFiles()
	broadcast(wantsHomePage, map[string]any{"type": "files_hosted_count", "data": fileCount})
}

// UpdateUserCount refreshes the user count and broadcasts it if it changed.
func UpdateUserCount() {
	local, err := database.GetCumulativeUserCount()
	if err != nil {
		fmt.Print("[hub] Error fetching user count: ", err)
		return
	}
	if local != userCount {
		userCount = local
		PublishUserCount()
	}
}

// PublishUserCount broadcasts the cached user count.
func PublishUserCount() {
	broadcast(wantsHomePage, map[string]any{"type": "user_count", "data": userCount})
}

// PublishSiteActivity records an activity timestamp and broadcasts the updated
// activity graph.
func PublishSiteActivity() {
	if !database.IsInitialized() {
		return
	}
	database.PushTimeStamp(time.Now().Unix())
	xAxis, yAxis := info.GetLastXDaysCounts()
	broadcast(wantsHomePage, map[string]any{
		"type": "graph_data",
		"data": protocol.GraphData{XAxis: xAxis, YAxis: yAxis, Label: "Site Activity", BeginAtZero: true},
	})
}

var (
	spaceMu      sync.Mutex
	lastXDays    [info.DaysTracked]string
	spaceUsedArr [info.DaysTracked]int64
)

// PublishSpaceUsed recomputes and broadcasts the space-used graph if it changed.
func PublishSpaceUsed() {
	newDays, newArr, err := info.GetSpaceUsedGraph()
	if err != nil {
		return
	}
	spaceMu.Lock()
	if lastXDays == newDays && spaceUsedArr == newArr {
		spaceMu.Unlock()
		return
	}
	lastXDays = newDays
	spaceUsedArr = newArr
	days, arr := lastXDays, spaceUsedArr
	spaceMu.Unlock()

	broadcast(wantsHomePage, map[string]any{
		"type": "graph_data",
		"data": protocol.GraphData{XAxis: days[:], YAxis: arr[:], Label: "Space Used", BeginAtZero: false},
	})
}

// PublishSysInfo broadcasts the current system information.
func PublishSysInfo(sysInfo interface{}) {
	broadcast(wantsHomePage, map[string]any{"type": "system_information", "data": sysInfo})
}

// SpaceUsedSnapshot returns the last broadcast space-used graph.
func SpaceUsedSnapshot() ([]string, []int64) {
	spaceMu.Lock()
	defer spaceMu.Unlock()
	return lastXDays[:], spaceUsedArr[:]
}

// StartTickers launches the periodic homepage broadcast loops.
func StartTickers() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			PublishSpaceUsed()
		}
	}()
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			sysInfo, err := info.GetSysInfo()
			if err != nil {
				continue
			}
			PublishSysInfo(sysInfo)
		}
	}()
}

// ForceLogout clears the session for every connection authenticated as email and
// notifies them.
func ForceLogout(email string) {
	var targets []globals.WebsocketInfo
	mu.Lock()
	for conn, data := range connections {
		if data.UserInfo.Email == email && data.UserInfo.HashedPassword != "" {
			d := data
			targets = append(targets, globals.WebsocketInfo{Conn: conn, Data: &d})
			d.UserInfo = globals.UserInfo{}
			connections[conn] = d
		}
	}
	mu.Unlock()
	for _, ci := range targets {
		go writeTo(ci, map[string]any{"type": "force_logout", "data": email})
	}
}
