package setup

import (
	"fmt"
	"net/http"
	"service/sysinfo"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// excuse the shitload of comments, i have no clue what im doing
func SetupWebsocket(r *gin.Engine) {
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

		stopUpdates := make(chan bool)
		done := make(chan bool)

		// this is where we handle the messages from the client
		// basically its a goroutiune that will asynchronously check if the client has requested to stop
		go handleClientMessages(conn, stopUpdates, done)
		// this is where we send the updates to the client
		// it will asynchronously send updates to the client
		go sendPeriodicUpdates(conn, stopUpdates)

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

func handleClientMessages(conn *websocket.Conn, stopUpdates chan bool, done chan bool) {
	defer func() { done <- true }()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}
		fmt.Printf("Received message: %s\n", msg)

		if string(msg) == "stop" {
			stopUpdates <- true
			break
		}
	}
}

func sendPeriodicUpdates(conn *websocket.Conn, stopUpdates chan bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sysinfo, _ := sysinfo.GetSysInfo()
			if err := conn.WriteJSON(sysinfo); err != nil {
				fmt.Println("Error sending update:", err)
				return
			}
		case <-stopUpdates:
			fmt.Println("Stopping updates.")
			return
		}
	}
}
