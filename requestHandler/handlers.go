package requestHandler

import (
	"angadrive/globals"
	"angadrive/hub"
	"angadrive/protocol"
	"angadrive/service"
	"encoding/json"

	"github.com/gorilla/websocket"
)

func handleGetUserFiles(conn *websocket.Conn, data json.RawMessage) {
	var req protocol.AuthInfo
	if err := json.Unmarshal(data, &req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "get_user_files_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}
	files, err := service.GetUserFiles(req)
	if err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "get_user_files_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}
	updateConnAuth(conn, req)
	hub.Send(conn, globals.OutgoingResponse{Type: "get_user_files_response", Data: files})
}

func handleGetUserCollections(conn *websocket.Conn, data json.RawMessage) {
	var req protocol.AuthInfo
	if err := json.Unmarshal(data, &req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "get_user_collections_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}
	collections, err := service.GetUserCollections(req)
	if err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "get_user_collections_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}
	updateConnAuth(conn, req)
	hub.Send(conn, globals.OutgoingResponse{Type: "get_user_collections_response", Data: collections})
}

func handleDeleteFile(conn *websocket.Conn, data json.RawMessage) {
	var req protocol.DeleteFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "delete_file_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}
	if deleted, err := service.DeleteFile(req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "delete_file_response", Data: map[string]interface{}{"error": err.Error()}})
	} else {
		hub.Send(conn, globals.OutgoingResponse{Type: "delete_file_response", Data: map[string]interface{}{"success": deleted.OriginalFileName}})
	}
}

func handleRenameFile(conn *websocket.Conn, data json.RawMessage) {
	var req protocol.RenameFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}
	updated, err := service.RenameFile(req)
	if err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}
	hub.Send(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"success": updated.OriginalFileName}})
}

func handleDuplicateFile(conn *websocket.Conn, data json.RawMessage) {
	var req protocol.DuplicateFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "duplicate_file_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}
	duplicate, err := service.DuplicateFile(req)
	if err != nil {
		hub.Send(conn, globals.OutgoingResponse{Type: "duplicate_file_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}
	hub.Send(conn, globals.OutgoingResponse{Type: "duplicate_file_response", Data: map[string]interface{}{"success": duplicate.OriginalFileName}})
}

func handleEnableHomepageUpdates(conn *websocket.Conn, data json.RawMessage) {
	var enabled bool
	if err := json.Unmarshal(data, &enabled); err != nil {
		return
	}
	hub.SetHomePageUpdates(conn, enabled)

	snapshot := service.BuildHomepageSnapshot()
	hub.Send(conn, map[string]interface{}{"type": "graph_data", "data": snapshot.SiteActivity})
	hub.Send(conn, map[string]interface{}{"type": "graph_data", "data": snapshot.SpaceUsed})
	hub.Send(conn, map[string]interface{}{"type": "user_count", "data": snapshot.UserCount})
	hub.Send(conn, map[string]interface{}{"type": "files_hosted_count", "data": snapshot.FileCount})
	hub.Send(conn, map[string]interface{}{"type": "system_information", "data": snapshot.SysInfo})
}
