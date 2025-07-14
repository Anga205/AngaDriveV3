package socketHandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"service/accounts"
	"service/database"
	"service/info"
	"service/vars"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	ActiveWebsockets      = make(map[*websocket.Conn]WebsocketData)
	ActiveWebsocketsMutex sync.RWMutex
	UPLOAD_DIR            string
)

func genericUserPulse(token string, message map[string]interface{}) {
	var connectionsToUpdate []connInfo
	ActiveWebsocketsMutex.RLock()
	for conn, connData := range ActiveWebsockets {
		connectionsToUpdate = append(connectionsToUpdate, connInfo{conn: conn, data: connData})
	}
	ActiveWebsocketsMutex.RUnlock()
	for _, ci := range connectionsToUpdate {
		go func(ci connInfo, token string) {
			if ci.data.UserInfo.Token == token {
				ci.data.Mutex.Lock()
				ci.conn.WriteJSON(message)
				ci.data.Mutex.Unlock()
			} else if ci.data.UserInfo.Email != "" && ci.data.UserInfo.HashedPassword != "" {
				if accounts.AuthenticateHashed(ci.data.UserInfo.Email, ci.data.UserInfo.HashedPassword) {
					user, _ := database.FindUserByEmail(ci.data.UserInfo.Email)
					if user.Token == token {
						ci.data.Mutex.Lock()
						ci.conn.WriteJSON(message)
						ci.data.Mutex.Unlock()
					}
				}
			}
		}(ci, token)
	}
}

func SetupWebsocket(r *gin.Engine, upload_dir string) {
	UPLOAD_DIR = upload_dir
	info.InitializeSysInfo()
	initializeUserCount()
	initFileCount()
	go initSpaceUsedPulser()
	go sysinfoPulse()
	r.GET("/ws", func(c *gin.Context) {
		if c.Request.Host != vars.BackendURL {
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
		ActiveWebsockets[conn] = WebsocketData{
			Mutex:                 &sync.Mutex{},
			HomePageUpdates:       false,
			UserInfo:              UserInfo{"", "", ""},
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
			fmt.Printf("Received message: %s\n", msg)
			var message IncomingMessage
			if err := json.Unmarshal(msg, &message); err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error unmarshalling JSON: %v\n", timestamp, err)
				continue
			}

			if handler, ok := messageHandlers[message.Type]; ok {
				rawData, _ := json.Marshal(message.Data)
				handler.Handle(conn, rawData)
			} else {
				fmt.Printf("Unknown message type: %s\n", message.Type)
			}
		}
	}
}

func updateConnAuth(conn *websocket.Conn, req AuthInfo) {
	ActiveWebsockets[conn].Mutex.Lock()
	defer ActiveWebsockets[conn].Mutex.Unlock()

	data := ActiveWebsockets[conn]
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
	ActiveWebsockets[conn] = data
}
