package globals

import (
	"sync"

	"github.com/gorilla/websocket"
)

type UserInfo struct {
	Token          string
	Email          string
	HashedPassword string
}

type WebsocketData struct {
	Mutex                 *sync.Mutex
	HomePageUpdates       bool
	UserInfo              UserInfo
	SubscribedCollections map[string]bool
}

type comms struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type IncomingMessage = comms

type OutgoingResponse = comms

type WebsocketInfo struct {
	Conn *websocket.Conn
	Data *WebsocketData
}
