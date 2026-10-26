// Package hub owns the live websocket connection registry and all realtime
// broadcast ("pulse") logic. Transports register connections here; the service
// layer publishes domain events here. Nothing in this package knows about HTTP
// routing or request parsing.
package hub

import (
	"angadrive/globals"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	connections = make(map[*websocket.Conn]globals.WebsocketData)
	mu          sync.RWMutex
)

// Register adds a new connection with empty session state.
func Register(conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()
	connections[conn] = globals.WebsocketData{
		Mutex:                 &sync.Mutex{},
		HomePageUpdates:       false,
		UserInfo:              globals.UserInfo{},
		SubscribedCollections: make(map[string]bool),
	}
}

// Unregister removes a connection from the registry.
func Unregister(conn *websocket.Conn) {
	mu.Lock()
	delete(connections, conn)
	mu.Unlock()
}

// Get returns a copy of the session state for a connection.
func Get(conn *websocket.Conn) (globals.WebsocketData, bool) {
	mu.RLock()
	defer mu.RUnlock()
	data, ok := connections[conn]
	return data, ok
}

// Update applies fn to the session state of a connection, if present.
func Update(conn *websocket.Conn, fn func(*globals.WebsocketData)) {
	mu.Lock()
	defer mu.Unlock()
	data, ok := connections[conn]
	if !ok {
		return
	}
	fn(&data)
	connections[conn] = data
}

// SetHomePageUpdates toggles homepage update delivery for a connection.
func SetHomePageUpdates(conn *websocket.Conn, enabled bool) {
	Update(conn, func(d *globals.WebsocketData) { d.HomePageUpdates = enabled })
}

// SubscribeCollection marks a connection as subscribed to a collection.
func SubscribeCollection(conn *websocket.Conn, collectionID string) {
	if collectionID == "" {
		return
	}
	Update(conn, func(d *globals.WebsocketData) { d.SubscribedCollections[collectionID] = true })
}

// Send writes a JSON message to a single connection, honoring the per-connection
// write mutex. Messages for unknown connections are dropped to preserve the
// single-writer contract.
func Send(conn *websocket.Conn, v interface{}) {
	mu.RLock()
	data, ok := connections[conn]
	mu.RUnlock()
	if !ok || data.Mutex == nil {
		fmt.Printf("Skipping websocket write: no per-connection state for %p\n", conn)
		return
	}
	data.Mutex.Lock()
	defer data.Mutex.Unlock()
	if err := conn.WriteJSON(v); err != nil {
		fmt.Printf("Error writing to websocket: %v\n", err)
	}
}

// snapshot returns copies of all connections matching filter.
func snapshot(filter func(globals.WebsocketData) bool) []globals.WebsocketInfo {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]globals.WebsocketInfo, 0, len(connections))
	for conn, data := range connections {
		if filter == nil || filter(data) {
			d := data
			out = append(out, globals.WebsocketInfo{Conn: conn, Data: &d})
		}
	}
	return out
}

// writeTo writes message to a connection and removes it on failure.
func writeTo(ci globals.WebsocketInfo, message interface{}) {
	ci.Data.Mutex.Lock()
	err := ci.Conn.WriteJSON(message)
	ci.Data.Mutex.Unlock()
	if err != nil {
		Unregister(ci.Conn)
		ci.Conn.Close()
	}
}

// broadcast sends message to every connection matching filter, each in its own
// goroutine.
func broadcast(filter func(globals.WebsocketData) bool, message interface{}) {
	for _, ci := range snapshot(filter) {
		go writeTo(ci, message)
	}
}

// broadcastEach sends a per-connection message built by build. If build returns
// ok=false the connection is skipped. build runs while holding the connection's
// write mutex so it may safely read session state.
func broadcastEach(filter func(globals.WebsocketData) bool, build func(globals.WebsocketData) (interface{}, bool)) {
	for _, ci := range snapshot(filter) {
		go func(ci globals.WebsocketInfo) {
			ci.Data.Mutex.Lock()
			message, ok := build(*ci.Data)
			if !ok {
				ci.Data.Mutex.Unlock()
				return
			}
			err := ci.Conn.WriteJSON(message)
			ci.Data.Mutex.Unlock()
			if err != nil {
				Unregister(ci.Conn)
				ci.Conn.Close()
			}
		}(ci)
	}
}

// hasCredentials reports whether a connection carries any auth information.
func hasCredentials(d globals.WebsocketData) bool {
	return (d.UserInfo.Email != "" && d.UserInfo.HashedPassword != "") || d.UserInfo.Token != ""
}

// wantsHomePage reports whether a connection subscribed to homepage updates.
func wantsHomePage(d globals.WebsocketData) bool { return d.HomePageUpdates }
