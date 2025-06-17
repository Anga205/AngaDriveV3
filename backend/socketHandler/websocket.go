package socketHandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"service/accounts"
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

func SetupWebsocket(r *gin.Engine) {
	info.InitializeSysInfo()
	go sysinfoPulse()
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
		go siteActivityPulse()
		defer conn.Close()

		ActiveWebsocketsMutex.Lock()
		ActiveWebsockets[conn] = WebsocketData{Mutex: &sync.Mutex{}, HomePageUpdates: false, UserInfo: UserInfo{"", "", ""}}
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
		var message IncomingMessage
		err = json.Unmarshal(msg, &message)
		if err != nil {
			now := time.Now()
			timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
			fmt.Printf("[%s] Error unmarshalling JSON: %v\n", timestamp, err)
			break
		}
		if message.Type == "register" {
			var req accounts.RegisterRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "register_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			}
			responseInfo, err := accounts.RegisterUser(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error registering user: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "register_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "register_response",
					Data: responseInfo,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "enable_homepage_updates" {
			ActiveWebsocketsMutex.Lock()
			data := ActiveWebsockets[conn]
			data.HomePageUpdates = message.Data.(bool)
			ActiveWebsockets[conn] = data
			ActiveWebsocketsMutex.Unlock()
			x_axis, y_axis := info.GetLast7DaysCounts()
			graphData := GraphData{
				XAxis:       x_axis,
				YAxis:       y_axis,
				Label:       "Site Activity",
				BeginAtZero: true,
			}
			ActiveWebsockets[conn].Mutex.Lock()
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
			ActiveWebsockets[conn].Mutex.Unlock()
		} else if message.Type == "login" {
			var req accounts.LoginRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "login_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			}
			responseInfo, err := accounts.LoginUser(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error logging in user: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "login_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "login_response",
					Data: responseInfo,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "change_password" {
			var req accounts.ChangePasswordRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_password_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			}
			responseInfo, err := accounts.ChangeUserPassword(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error changing password: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_password_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_password_response",
					Data: responseInfo,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "change_email" {
			var req accounts.ChangeEmailRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_email_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			}
			responseInfo, err := accounts.ChangeUserEmail(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error changing email: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_email_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_email_response",
					Data: responseInfo,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "change_display_name" {
			var req accounts.ChangeDisplayNameRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_display_name_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			}
			responseInfo, err := accounts.ChangeUserDisplayName(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error changing display name: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_display_name_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_display_name_response",
					Data: responseInfo,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "get_user_files" {
			var req AuthRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "get_user_files_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			}
			files, err := GetUserFiles(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error getting user files: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "get_user_files_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				return
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "get_user_files_response",
					Data: files,
				})
				if req.Email != "" && req.Password != "" && accounts.Authenticate(req.Email, req.Password) {
					accountInfo, _ := database.FindUserByEmail(req.Email)
					data := ActiveWebsockets[conn]
					data.UserInfo.Token = ""
					data.UserInfo.Email = accountInfo.Email
					data.UserInfo.HashedPassword = accountInfo.HashedPassword
					ActiveWebsockets[conn] = data
				} else {
					data := ActiveWebsockets[conn]
					data.UserInfo.Token = req.Token
					data.UserInfo.Email = ""
					data.UserInfo.HashedPassword = ""
					ActiveWebsockets[conn] = data
				}
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else {
			fmt.Printf("Unknown message type: %s\n", message.Type)
			continue
		}
	}
}
