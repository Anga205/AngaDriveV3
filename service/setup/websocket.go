package setup

import (
	"fmt"
	"net/http"
	"service/sysinfo"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	activeWebsockets      = make(map[*websocket.Conn]bool)
	activeWebsocketsMutex sync.RWMutex
)

func updateAllWebsockets() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		sysInfo, _ := sysinfo.GetSysInfo()

		for conn := range activeWebsockets {
			err := conn.WriteJSON(map[string]interface{}{
				"type": "system_information",
				"data": sysInfo,
			})
			if err != nil {
				fmt.Printf("Error writing to websocket: %v\n", err)
				activeWebsocketsMutex.Lock()
				delete(activeWebsockets, conn)
				activeWebsocketsMutex.Unlock()
				conn.Close()
			}
		}
	}
}

// excuse the shitload of comments, i have no clue what im doing
func SetupWebsocket(r *gin.Engine) {
	sysinfo.InitializeSysInfo()
	go updateAllWebsockets()
	r.GET("/ws", func(c *gin.Context) {
		// Upgrade the connection to a websocket
		conn, err := upgradeToWebSocket(c)
		if err != nil {
			// if we fail to upgrade, we just return a 500 error
			// i've never actually encountered a situation where this block gets executed
			// but i guess its good practice to have it
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to websocket"})
			return
		}

		// defer just means "when this function exits, close the connection"
		defer conn.Close()

		// add the connection to the list of active websockets
		activeWebsocketsMutex.Lock()
		activeWebsockets[conn] = true
		activeWebsocketsMutex.Unlock()

		defer func() {
			// remove the connection from the list of active websockets
			activeWebsocketsMutex.Lock()
			delete(activeWebsockets, conn)
			activeWebsocketsMutex.Unlock()
		}()

		done := make(chan bool)

		// this is where we handle the messages from the client
		// basically its a goroutiune that will asynchronously check if the client has requested to stop
		go handleClose(conn, done)

		// this is where we wait for the client to stop
		// its an empty channel that only gets pushed to when the client has requested to stop
		<-done
	})
}

func upgradeToWebSocket(c *gin.Context) (*websocket.Conn, error) {
	// this just defines the upgrade to websocket
	upgrader := websocket.Upgrader{
		// for now, we just allow all origins
		// later we can add a check for the origin
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO: add a check for the origin
		},
	}
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

func handleClose(conn *websocket.Conn, done chan bool) {
	defer func() { done <- true }()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			break
		}
		fmt.Printf("Received message: %s\n", msg)
	}
}
