package requestHandler

import (
	"angadrive/accounts"
	"angadrive/globals"
	"angadrive/hub"
	"angadrive/protocol"
	"angadrive/service"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// authInfo is the wire form of protocol.AuthInfo used by handlers that need to
// refresh the connection session after authenticating.
type authInfo = protocol.AuthInfo

// processRequest decodes data into T, runs handler, and sends the result as a
// response of type responseType. It centralizes error formatting so handlers
// stay focused on business calls.
func processRequest[T any, R any](conn *websocket.Conn, data json.RawMessage, handler func(T) (R, error), responseType string) {
	var req T
	if err := json.Unmarshal(data, &req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: responseType, Data: map[string]string{"error": "invalid request data"}})
		return
	}

	responseInfo, err := handler(req)
	if err != nil {
		fmt.Printf("[%s] Error handling %s: %v\n", time.Now().Format("03:04:05 PM, 02 Jan 2006"), responseType, err)
		if errors.Is(err, accounts.ErrStaleClient) {
			hub.Send(conn, globals.OutgoingResponse{Type: "notification", Data: accounts.StaleClientMessage})
		}
		errorType := "error"
		if responseType == "login_response" || responseType == "migrate_password_response" {
			errorType = responseType
		}
		hub.Send(conn, globals.OutgoingResponse{Type: errorType, Data: err.Error()})
		return
	}

	if responseType == "get_collection_response" {
		if r, ok := any(req).(interface{ GetCollectionID() string }); ok {
			hub.SubscribeCollection(conn, r.GetCollectionID())
		}
	}
	hub.Send(conn, globals.OutgoingResponse{Type: responseType, Data: responseInfo})
}

// dispatchMessage routes an incoming websocket message to its handler. A switch
// is used so every handler call is statically type-checked.
func dispatchMessage(conn *websocket.Conn, messageType string, data json.RawMessage) {
	switch messageType {
	case "register":
		processRequest(conn, data, accounts.RegisterUser, "register_response")
		go hub.UpdateUserCount()
	case "login":
		processRequest(conn, data, accounts.LoginUser, "login_response")
	case "migrate_password":
		processRequest(conn, data, accounts.MigratePassword, "migrate_password_response")
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
		processRequest(conn, data, service.HandleConversionRequest, "convert_video_response")
	case "convert_image":
		processRequest(conn, data, service.HandleImageConversionRequest, "convert_image_response")
	case "delete_file":
		handleDeleteFile(conn, data)
	case "rename_file":
		handleRenameFile(conn, data)
	case "duplicate_file":
		handleDuplicateFile(conn, data)
	case "bulk_duplicate_files":
		processRequest(conn, data, service.BulkDuplicateFile, "bulk_duplicate_files_response")
	case "bulk_delete_files":
		processRequest(conn, data, service.BulkDeleteFile, "bulk_delete_files_response")
	case "bulk_add_files_to_collection":
		processRequest(conn, data, service.BulkAddFilesToCollection, "bulk_add_files_to_collection_response")
	case "new_collection":
		processRequest(conn, data, service.CreateNewCollection, "new_collection_response")
	case "delete_collection":
		processRequest(conn, data, service.DeleteCollection, "delete_collection_response")
	case "get_collection":
		processRequest(conn, data, service.GetCollection, "get_collection_response")
	case "enable_homepage_updates":
		handleEnableHomepageUpdates(conn, data)
	case "add_folder_to_collection":
		processRequest(conn, data, service.AddFolder, "get_collection_response")
	case "remove_folder_from_collection":
		processRequest(conn, data, service.RemoveFolder, "get_collection_response")
	case "create_folder_in_collection":
		processRequest(conn, data, service.CreateFolderInCollection, "get_collection_response")
	case "add_file_to_collection":
		processRequest(conn, data, service.AddFileToCollection, "get_collection_response")
	case "remove_file_from_collection":
		processRequest(conn, data, service.RemoveFileFromCollection, "get_collection_response")
	case "import_from_github":
		// Run off the read loop: imports can take minutes.
		go processRequest(conn, data, service.GithubImportHandler, "success_notification")
	case "delete_account":
		processRequest(conn, data, service.RemoveAccount, "success_notification")
	default:
		fmt.Printf("Unknown message type: %s\n", messageType)
	}
}
