package setup

import "sync"

type WebsocketData struct {
	Mutex           *sync.Mutex
	HomePageUpdates bool
}

type IncomingMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type OutgoingResponse struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
