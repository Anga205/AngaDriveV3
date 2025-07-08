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
				continue
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
				continue
			} else {
				go UpdateUserCount()
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
			siteActivityData := GraphData{
				XAxis:       x_axis,
				YAxis:       y_axis,
				Label:       "Site Activity",
				BeginAtZero: true,
			}
			sysinfo, _ := info.GetSysInfo()
			ActiveWebsockets[conn].Mutex.Lock()
			conn.WriteJSON(map[string]interface{}{
				"type": "graph_data",
				"data": siteActivityData,
			})
			conn.WriteJSON(map[string]interface{}{
				"type": "graph_data",
				"data": GraphData{
					XAxis:       Last7Days[:],
					YAxis:       SpaceUsedArr[:],
					Label:       "Space Used",
					BeginAtZero: false,
				},
			})
			conn.WriteJSON(map[string]interface{}{
				"type": "user_count",
				"data": userCount,
			})
			conn.WriteJSON(map[string]interface{}{
				"type": "files_hosted_count",
				"data": fileCount,
			})
			err := conn.WriteJSON(map[string]interface{}{
				"type": "system_information",
				"data": sysinfo,
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
				continue
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
				continue
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
				continue
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
				continue
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
				continue
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
				continue
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
				continue
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
				continue
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "change_display_name_response",
					Data: responseInfo,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "get_user_files" {
			var req AuthInfo
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "get_user_files_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
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
				continue
			} else {
				ActiveWebsockets[conn].Mutex.Lock()
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
				conn.WriteJSON(OutgoingResponse{
					Type: "get_user_files_response",
					Data: files,
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
		} else if message.Type == "convert_video" {
			var req ConvertVideoRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "convert_video_response",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
			}
			if req.Auth.Token == "" {
				if !accounts.Authenticate(req.Auth.Email, req.Auth.Password) {
					now := time.Now()
					timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
					fmt.Printf("[%s] Authentication failed for convert_video request\n", timestamp)
					ActiveWebsockets[conn].Mutex.Lock()
					conn.WriteJSON(OutgoingResponse{
						Type: "convert_video_response",
						Data: map[string]interface{}{
							"error": "authentication failed",
						},
					})
					ActiveWebsockets[conn].Mutex.Unlock()
					continue
				}
				user, _ := database.FindUserByEmail(req.Auth.Email)
				req.Auth.Token = user.Token
			}
			fileDirectory := req.FileDirectory
			fileToConvert, err := database.GetFile(fileDirectory)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error fetching file: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "convert_video_response",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			if fileToConvert.AccountToken != req.Auth.Token {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Authentication failed for convert_video request: file %s does not belong to user %s\n", timestamp, fileToConvert.OriginalFileName, req.Auth.Email)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "convert_video_response",
					Data: map[string]interface{}{
						"error": "authentication failed: file does not belong to user",
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			go ConvertToMP4(fileToConvert, ActiveWebsockets[conn].Mutex, conn)
		} else if message.Type == "delete_file" {
			var req DeleteFileRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			fileToDelete, _ := database.GetFile(req.FileDirectory)
			err = DeleteFile(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error deleting file: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			go UserFilesPulse(FileUpdate{
				Toggle: false,
				File:   fileToDelete,
			})
			go UpdateUserCount()
			ActiveWebsockets[conn].Mutex.Lock()
			conn.WriteJSON(OutgoingResponse{
				Type: "delete_file_response",
				Data: map[string]interface{}{
					"success": fileToDelete.OriginalFileName,
				},
			})
			ActiveWebsockets[conn].Mutex.Unlock()
		} else if message.Type == "new_collection" {
			var req CreateCollectionRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			err = CreateNewCollection(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error creating collection: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
		} else if message.Type == "delete_collection" {
			var req DeleteCollectionRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			err = DeleteCollection(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error deleting collection: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
		} else if message.Type == "get_user_collections" {
			var req AuthInfo
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			collections, err := GetUserCollections(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error getting user collections: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			ActiveWebsockets[conn].Mutex.Lock()
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
			if collections == nil {
				collections = []CollectionCardData{} // not doing this will cause the frontend to crash, as nil evaluates to null in JSON
			}
			conn.WriteJSON(OutgoingResponse{
				Type: "get_user_collections_response",
				Data: collections,
			})
			ActiveWebsockets[conn].Mutex.Unlock()
		} else if message.Type == "get_collection" {
			var req GetCollectionRequest
			dataBytes, _ := json.Marshal(message.Data)
			err := json.Unmarshal(dataBytes, &req)
			if err != nil {
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{"error": "invalid request data"},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			response, err := GetCollection(req)
			if err != nil {
				now := time.Now()
				timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
				fmt.Printf("[%s] Error getting collection: %v\n", timestamp, err)
				ActiveWebsockets[conn].Mutex.Lock()
				conn.WriteJSON(OutgoingResponse{
					Type: "error",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				})
				ActiveWebsockets[conn].Mutex.Unlock()
				continue
			}
			ActiveWebsockets[conn].Mutex.Lock()
			conn.WriteJSON(OutgoingResponse{
				Type: "get_collection_response",
				Data: response,
			})
			ActiveWebsockets[conn].Mutex.Unlock()
		} else {
			fmt.Printf("Unknown message type: %s\n", message.Type)
			continue
		}
	}
}
