package setup

import (
	"fmt"
	"net/http"
	"service/database"
	"service/info"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	ActiveWebsockets      = make(map[*websocket.Conn]bool)
	ActiveWebsocketsMutex sync.RWMutex
)

type GraphData struct {
	XAxis       []string `json:"x_axis"`
	YAxis       []int64  `json:"y_axis"`
	Label       string   `json:"label"`
	BeginAtZero bool     `json:"begin_at_zero"`
}

func Pulse() {
	database.PushTimeStamp(time.Now().Unix())
	x_axis, y_axis := info.GetLast7DaysCounts()
	graphData := GraphData{
		XAxis:       x_axis,
		YAxis:       y_axis,
		Label:       "Site Activity",
		BeginAtZero: true,
	}
	for conn := range ActiveWebsockets {
		err := conn.WriteJSON(map[string]interface{}{
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
	}
}

func updateAllWebsockets() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		sysInfo, _ := info.GetSysInfo()

		for conn := range ActiveWebsockets {
			err := conn.WriteJSON(map[string]interface{}{
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
		}
	}
}

// excuse the shitload of comments, i have no clue what im doing
func SetupWebsocket(r *gin.Engine) {
	info.InitializeSysInfo()
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

		// give the client some initial data before the ticker takes over
		sysInfo, _ := info.GetSysInfo()
		conn.WriteJSON(map[string]interface{}{
			"type": "system_information",
			"data": sysInfo,
		})
		go Pulse()
		// defer just means "when this function exits, close the connection"
		defer conn.Close()

		// add the connection to the list of active websockets
		ActiveWebsocketsMutex.Lock()
		ActiveWebsockets[conn] = true
		ActiveWebsocketsMutex.Unlock()

		defer func() {
			// remove the connection from the list of active websockets
			ActiveWebsocketsMutex.Lock()
			delete(ActiveWebsockets, conn)
			ActiveWebsocketsMutex.Unlock()
		}()

		done := make(chan bool)

		// this is where we handle the messages from the client
		// basically its a goroutiune that will asynchronously check if the client has requested to stop
		go handleClose(conn, done)

		// this is an empty channel that only gets pushed to when the client has requested to stop
		// this is needed because it delays the function from exiting therefore delaying the closing of the connection
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
