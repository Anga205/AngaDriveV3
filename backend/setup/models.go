package setup

import "sync"

type WebsocketData struct {
	Mutex     *sync.Mutex
	UserToken string
}
