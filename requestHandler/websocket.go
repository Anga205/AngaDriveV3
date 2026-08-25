package requestHandler

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/globals"
	"angadrive/info"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	ActiveWebsockets      = make(map[*websocket.Conn]globals.WebsocketData)
	ActiveWebsocketsMutex sync.RWMutex
)

func genericUserPulse(token string, message map[string]interface{}) {
	var connectionsToUpdate []globals.WebsocketInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		connectionsToUpdate = append(connectionsToUpdate, globals.WebsocketInfo{Conn: conn, Data: &connData})
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(ci globals.WebsocketInfo, token string) {
			if ci.Data.UserInfo.Token == token {
				ci.Data.Mutex.Lock()
				ci.Conn.WriteJSON(message)
				ci.Data.Mutex.Unlock()
			} else if ci.Data.UserInfo.Email != "" && ci.Data.UserInfo.HashedPassword != "" {
				if accounts.AuthenticateHashed(ci.Data.UserInfo.Email, ci.Data.UserInfo.HashedPassword) {
					user, _ := database.FindUserByEmail(ci.Data.UserInfo.Email)
					if user.Token == token {
						ci.Data.Mutex.Lock()
						ci.Conn.WriteJSON(message)
						ci.Data.Mutex.Unlock()
					}
				}
			}
		}(ci, token)
	}
}

func SetupWebsocket(r *gin.Engine) {
	initRunners()
	info.InitializeSysInfo()
	initializeUserCount()
	initFileCount()
	go initSpaceUsedPulser()
	go sysinfoPulse()
	r.GET("/ws", func(c *gin.Context) {
		if c.Request.Host != globals.WebURL {
			// if gin.Mode() != gin.ReleaseMode {
			// 	fmt.Printf("Websocket connection attempt from disallowed host: %s\n", c.Request.Host)
			// 	fmt.Printf("Try with host: %s\n", globals.WebURL)
			// } // TODO: redo the env var checking
			c.JSON(http.StatusForbidden, gin.H{"error": "Websocket connection not allowed from this host"})
			return
		}
		conn, err := upgradeToWebSocket(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to websocket"})
			return
		}
		defer conn.Close()

		ActiveWebsocketsMutex.Lock()
		ActiveWebsockets[conn] = globals.WebsocketData{
			Mutex:                 &sync.Mutex{},
			HomePageUpdates:       false,
			UserInfo:              globals.UserInfo{Email: "", Token: "", HashedPassword: ""},
			SubscribedCollections: make(map[string]bool),
		}
		ActiveWebsocketsMutex.Unlock()

		defer func() {
			ActiveWebsocketsMutex.Lock()
			delete(ActiveWebsockets, conn)
			ActiveWebsocketsMutex.Unlock()
		}()

		done := make(chan bool)
		go reader(conn, done)
		<-done
	})
}

func upgradeToWebSocket(c *gin.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

func reader(conn *websocket.Conn, done chan bool) {
	defer func() { done <- true }()
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("Error reading message: %v\n", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			var message globals.IncomingMessage
			if err := json.Unmarshal(msg, &message); err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error unmarshalling JSON: %v\n", timestamp, err)
				continue
			}

			rawData, _ := json.Marshal(message.Data)
			dispatchMessage(conn, message.Type, rawData)
		}
	}
}

func updateConnAuth(conn *websocket.Conn, req AuthInfo) {
	ActiveWebsocketsMutex.RLock()
	data, ok := ActiveWebsockets[conn]
	ActiveWebsocketsMutex.RUnlock()
	if !ok || data.Mutex == nil {
		// Connection state no longer available; nothing to update
		return
	}
	data.Mutex.Lock()
	defer data.Mutex.Unlock()
	if req.Email != "" && req.Password != "" && accounts.Authenticate(req.Email, req.Password) {
		accountInfo, _ := database.FindUserByEmail(req.Email)
		data.UserInfo.Token = ""
		data.UserInfo.Email = accountInfo.Email
		data.UserInfo.HashedPassword = accountInfo.HashedPassword
	} else {
		data.UserInfo.Token = req.Token
		data.UserInfo.Email = ""
		data.UserInfo.HashedPassword = ""
	}
	ActiveWebsocketsMutex.Lock()
	// Double-check still present before writing back
	if _, stillOK := ActiveWebsockets[conn]; stillOK {
		ActiveWebsockets[conn] = data
	}
	ActiveWebsocketsMutex.Unlock()
}
