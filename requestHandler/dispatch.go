package requestHandler

import (
	"angadrive/accounts"
	"angadrive/globals"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// processRequest is a generic function to handle common request-response patterns.
func processRequest[T any, R any](conn *websocket.Conn, data json.RawMessage, handler func(T) (R, error), responseType string) {
	var req T
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{
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
		// Login errors are sent back with the login response type so the client
		// can surface them in the login form.
		errorType := "error"
		if responseType == "login_response" {
			errorType = "login_response"
		}
		sendJSON(conn, globals.OutgoingResponse{
			Type: errorType,
			Data: err.Error(),
		})
		return
	}

	if responseType == "get_collection_response" {
		if r, ok := any(req).(interface{ GetCollectionID() string }); ok {
			collectionID := r.GetCollectionID()
			if collectionID != "" {
				ActiveWebsocketsMutex.Lock()
				wsData := ActiveWebsockets[conn]
				wsData.SubscribedCollections[collectionID] = true
				ActiveWebsockets[conn] = wsData
				ActiveWebsocketsMutex.Unlock()
			}
		}
	}
	sendJSON(conn, globals.OutgoingResponse{
		Type: responseType,
		Data: responseInfo,
	})
}

// sendJSON safely sends a JSON message to a websocket connection.
func sendJSON(conn *websocket.Conn, v interface{}) {
	// Safely read the connection data; the conn may have been removed already
	ActiveWebsocketsMutex.RLock()
	wsData, ok := ActiveWebsockets[conn]
	defer ActiveWebsocketsMutex.RUnlock()

	if ok && wsData.Mutex != nil {
		wsData.Mutex.Lock()
		defer wsData.Mutex.Unlock()
		if err := conn.WriteJSON(v); err != nil {
			fmt.Printf("Error writing to websocket: %v\n", err)
		}
		return
	}
	// No per-connection state: drop the message to avoid concurrent writes without a mutex.
	// This situation typically means the connection is closing/closed or was not initialized yet.
	// Intentionally NOT writing to the socket here to preserve the single-writer contract.
	fmt.Printf("Skipping websocket write: no per-connection state for %p\n", conn)
}

// dispatchMessage routes an incoming websocket message to its handler based on
// the message type. A switch is used instead of a map so that every handler
// call is statically type-checked against its request type.
func dispatchMessage(conn *websocket.Conn, messageType string, data json.RawMessage) {
	switch messageType {
	case "register":
		processRequest(conn, data, accounts.RegisterUser, "register_response")
		go UpdateUserCount()
	case "login":
		processRequest(conn, data, accounts.LoginUser, "login_response")
	case "change_password":
		processRequest(conn, data, accounts.ChangeUserPassword, "change_password_response")
	case "change_email":
		processRequest(conn, data, accounts.ChangeUserEmail, "change_email_response")
	case "change_display_name":
		processRequest(conn, data, accounts.ChangeUserDisplayName, "change_display_name_response")
	case "get_user_files":
		handleGetUserFiles(conn, data)
	case "get_user_collections":
		handleGetUserCollections(conn, data)
	case "convert_video":
		processRequest(conn, data, HandleConversionRequest, "convert_video_response")
	case "convert_image":
		processRequest(conn, data, HandleImageConversionRequest, "convert_image_response")
	case "delete_file":
		handleDeleteFile(conn, data)
	case "rename_file":
		handleRenameFile(conn, data)
	case "duplicate_file":
		handleDuplicateFile(conn, data)
	case "bulk_duplicate_files":
		processRequest(conn, data, BulkDuplicateFile, "bulk_duplicate_files_response")
	case "bulk_delete_files":
		processRequest(conn, data, BulkDeleteFile, "bulk_delete_files_response")
	case "bulk_add_files_to_collection":
		processRequest(conn, data, BulkAddFilesToCollection, "bulk_add_files_to_collection_response")
	case "new_collection":
		processRequest(conn, data, CreateNewCollection, "new_collection_response")
	case "delete_collection":
		processRequest(conn, data, DeleteCollection, "delete_collection_response")
	case "get_collection":
		processRequest(conn, data, GetCollection, "get_collection_response")
	case "enable_homepage_updates":
		handleEnableHomepageUpdates(conn, data)
	case "add_folder_to_collection":
		processRequest(conn, data, AddFolder, "get_collection_response")
	case "remove_folder_from_collection":
		processRequest(conn, data, RemoveFolder, "get_collection_response")
	case "create_folder_in_collection":
		processRequest(conn, data, CreateFolderInCollection, "get_collection_response")
	case "add_file_to_collection":
		processRequest(conn, data, AddFileToCollection, "get_collection_response")
	case "remove_file_from_collection":
		processRequest(conn, data, RemoveFileFromCollection, "get_collection_response")
	case "import_from_github":
		processRequest(conn, data, GithubImportHandler, "success_notification")
	case "delete_account":
		processRequest(conn, data, removeAccountHandler, "success_notification")
	default:
		fmt.Printf("Unknown message type: %s\n", messageType)
	}
}
