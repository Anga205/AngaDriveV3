package socketHandler

import (
	"encoding/json"
	"fmt"
	"service/accounts"
	"service/database"
	"service/info"
	"time"

	"github.com/gorilla/websocket"
)

// MessageHandler defines a generic interface for handling websocket messages.
type MessageHandler interface {
	Handle(*websocket.Conn, json.RawMessage)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as MessageHandlers.
type HandlerFunc func(*websocket.Conn, json.RawMessage)

// Handle calls f(conn, data).
func (f HandlerFunc) Handle(conn *websocket.Conn, data json.RawMessage) {
	f(conn, data)
}

// processRequest is a generic function to handle common request-response patterns.
func processRequest[T any, R any](conn *websocket.Conn, data json.RawMessage, handler func(T) (R, error), responseType string) {
	var req T
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: responseType,
			Data: map[string]string{"error": "invalid request data"},
		})
		return
	}

	responseInfo, err := handler(req)
	if err != nil {
		now := time.Now()
		timestamp := now.Format("03:04:05 PM, 02 Jan 2006")
		fmt.Printf("[%s] Error handling %s: %v\n", timestamp, responseType, err)
		sendJSON(conn, OutgoingResponse{
			Type: "error",
			Data: map[string]string{"error": err.Error()},
		})
		return
	}

	sendJSON(conn, OutgoingResponse{
		Type: responseType,
		Data: responseInfo,
	})
}

// sendJSON safely sends a JSON message to a websocket connection.
func sendJSON(conn *websocket.Conn, v interface{}) {
	ActiveWebsockets[conn].Mutex.Lock()
	defer ActiveWebsockets[conn].Mutex.Unlock()
	if err := conn.WriteJSON(v); err != nil {
		fmt.Printf("Error writing to websocket: %v\n", err)
	}
}

// messageHandlers maps message types to their handlers.
var messageHandlers = map[string]MessageHandler{
	"register": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, accounts.RegisterUser, "register_response")
		go UpdateUserCount()
	}),
	"login": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, accounts.LoginUser, "login_response")
	}),
	"change_password": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, accounts.ChangeUserPassword, "change_password_response")
	}),
	"change_email": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, accounts.ChangeUserEmail, "change_email_response")
	}),
	"change_display_name": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, accounts.ChangeUserDisplayName, "change_display_name_response")
	}),
	"get_user_files":       HandlerFunc(handleGetUserFiles),
	"get_user_collections": HandlerFunc(handleGetUserCollections),
	"convert_video": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, HandleConversionRequest, "convert_video_response")
	}),
	"delete_file": HandlerFunc(handleDeleteFile),
	"new_collection": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, CreateNewCollection, "new_collection_response")
	}),
	"delete_collection": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, DeleteCollection, "delete_collection_response")
	}),
	"get_collection": HandlerFunc(func(conn *websocket.Conn, data json.RawMessage) {
		processRequest(conn, data, GetCollection, "get_collection_response")
	}),
	"enable_homepage_updates": HandlerFunc(handleEnableHomepageUpdates),
}

func handleEnableHomepageUpdates(conn *websocket.Conn, data json.RawMessage) {
	var enabled bool
	if err := json.Unmarshal(data, &enabled); err != nil {
		return
	}

	ActiveWebsocketsMutex.Lock()
	wsData := ActiveWebsockets[conn]
	wsData.HomePageUpdates = enabled
	ActiveWebsockets[conn] = wsData
	ActiveWebsocketsMutex.Unlock()

	x_axis, y_axis := info.GetLast7DaysCounts()
	siteActivityData := GraphData{
		XAxis:       x_axis,
		YAxis:       y_axis,
		Label:       "Site Activity",
		BeginAtZero: true,
	}
	sysinfo, _ := info.GetSysInfo()

	sendJSON(
		conn,
		map[string]interface{}{
			"type": "graph_data",
			"data": siteActivityData,
		})
	sendJSON(conn, map[string]interface{}{
		"type": "graph_data",
		"data": GraphData{XAxis: Last7Days[:], YAxis: SpaceUsedArr[:], Label: "Space Used", BeginAtZero: false},
	})
	sendJSON(conn, map[string]interface{}{
		"type": "user_count",
		"data": userCount,
	})
	sendJSON(conn, map[string]interface{}{
		"type": "files_hosted_count",
		"data": fileCount,
	})
	sendJSON(conn, map[string]interface{}{
		"type": "system_information",
		"data": sysinfo,
	})
}

func handleGetUserFiles(conn *websocket.Conn, data json.RawMessage) {
	var req AuthInfo
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: "get_user_files_response",
			Data: map[string]interface{}{"error": "invalid request data"},
		})
		return
	}
	files, err := GetUserFiles(req)
	if err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: "get_user_files_response",
			Data: map[string]interface{}{"error": err.Error()},
		})
		return
	}
	updateConnAuth(conn, req)
	sendJSON(conn, OutgoingResponse{Type: "get_user_files_response", Data: files})
}

func handleGetUserCollections(conn *websocket.Conn, data json.RawMessage) {
	var req AuthInfo
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: "get_user_collections_response",
			Data: map[string]interface{}{"error": "invalid request data"},
		})
		return
	}
	collections, err := GetUserCollections(req)
	if err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: "get_user_collections_response",
			Data: map[string]interface{}{"error": err.Error()},
		})
		return
	}
	updateConnAuth(conn, req)
	sendJSON(conn, OutgoingResponse{Type: "get_user_collections_response", Data: collections})
}

func handleDeleteFile(conn *websocket.Conn, data json.RawMessage) {
	var req DeleteFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: "delete_file_response",
			Data: map[string]interface{}{"error": "invalid request data"},
		})
		return
	}

	fileToDelete, _ := database.GetFile(req.FileDirectory)
	if err := DeleteFile(req); err != nil {
		sendJSON(conn, OutgoingResponse{
			Type: "delete_file_response",
			Data: map[string]interface{}{"error": err.Error()},
		})
		return
	}

	go UserFilesPulse(FileUpdate{
		Toggle: false,
		File:   fileToDelete,
	})
	go UpdateUserCount()
	sendJSON(conn, OutgoingResponse{
		Type: "delete_file_response",
		Data: map[string]interface{}{"success": fileToDelete.OriginalFileName},
	})
}
