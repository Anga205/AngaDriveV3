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
	ActiveWebsockets      = make(map[*websocket.Conn]WebsocketData)
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
	for conn, connMutex := range ActiveWebsockets {
		connMutex.Mutex.Lock()
		err := conn.WriteJSON(map[string]interface{}{
			"type": "graph_data",
			"data": graphData,
		})
		connMutex.Mutex.Unlock()
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

		for conn, connMutex := range ActiveWebsockets {
			connMutex.Mutex.Lock()
			err := conn.WriteJSON(map[string]interface{}{
				"type": "system_information",
				"data": sysInfo,
			})
			connMutex.Mutex.Unlock()
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

func SetupWebsocket(r *gin.Engine) {
	info.InitializeSysInfo()
	go updateAllWebsockets()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgradeToWebSocket(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to websocket"})
			return
		}

		sysInfo, _ := info.GetSysInfo()
		conn.WriteJSON(map[string]interface{}{
			"type": "system_information",
			"data": sysInfo,
		})
		go Pulse()
		defer conn.Close()

		ActiveWebsocketsMutex.Lock()
		ActiveWebsockets[conn] = WebsocketData{Mutex: &sync.Mutex{}}
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
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			break
		}
		fmt.Printf("Received message: %s\n", msg)
	}
}
