package requestHandler

import (
	"angadrive/database"
	"angadrive/globals"
	"encoding/json"

	"github.com/gorilla/websocket"
)

func handleGetUserFiles(conn *websocket.Conn, data json.RawMessage) {
	var req AuthInfo
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{
			Type: "get_user_files_response",
			Data: map[string]interface{}{"error": "invalid request data"},
		})
		return
	}
	files, err := GetUserFiles(req)
	if err != nil {
		sendJSON(conn, globals.OutgoingResponse{
			Type: "get_user_files_response",
			Data: map[string]interface{}{"error": err.Error()},
		})
		return
	}
	updateConnAuth(conn, req)
	sendJSON(conn, globals.OutgoingResponse{Type: "get_user_files_response", Data: files})
}

func handleGetUserCollections(conn *websocket.Conn, data json.RawMessage) {
	var req AuthInfo
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{
			Type: "get_user_collections_response",
			Data: map[string]interface{}{"error": "invalid request data"},
		})
		return
	}
	collections, err := GetUserCollections(req)
	if err != nil {
		sendJSON(conn, globals.OutgoingResponse{
			Type: "get_user_collections_response",
			Data: map[string]interface{}{"error": err.Error()},
		})
		return
	}
	updateConnAuth(conn, req)
	sendJSON(conn, globals.OutgoingResponse{Type: "get_user_collections_response", Data: collections})
}

func handleDeleteFile(conn *websocket.Conn, data json.RawMessage) {
	var req DeleteFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{
			Type: "delete_file_response",
			Data: map[string]interface{}{"error": "invalid request data"},
		})
		return
	}

	fileToDelete, _ := database.GetFile(req.FileDirectory)
	if err := DeleteFile(req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{
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
	sendJSON(conn, globals.OutgoingResponse{
		Type: "delete_file_response",
		Data: map[string]interface{}{"success": fileToDelete.OriginalFileName},
	})
}

func handleRenameFile(conn *websocket.Conn, data json.RawMessage) {
	var req RenameFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}

	token, err := req.Auth.GetToken()
	if err != nil {
		sendJSON(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}
	file, err := database.GetFile(req.FileDirectory)
	if err != nil {
		sendJSON(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": "file not found"}})
		return
	}
	if file.AccountToken != token {
		sendJSON(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": "unauthorized rename attempt"}})
		return
	}
	updatedFile, err := database.RenameFile(req.FileDirectory, req.NewFileName)
	if err != nil {
		sendJSON(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}

	update := FileUpdate{Toggle: true, Replace: true, File: updatedFile}
	go UserFilesPulse(update)
	go PulseFileSubscribers(updatedFile)
	sendJSON(conn, globals.OutgoingResponse{Type: "rename_file_response", Data: map[string]interface{}{"success": updatedFile.OriginalFileName}})
}

func handleDuplicateFile(conn *websocket.Conn, data json.RawMessage) {
	var req DuplicateFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendJSON(conn, globals.OutgoingResponse{Type: "duplicate_file_response", Data: map[string]interface{}{"error": "invalid request data"}})
		return
	}

	duplicatedFile, err := DuplicateFile(req)
	if err != nil {
		sendJSON(conn, globals.OutgoingResponse{Type: "duplicate_file_response", Data: map[string]interface{}{"error": err.Error()}})
		return
	}

	go UserFilesPulse(FileUpdate{Toggle: true, File: duplicatedFile})
	sendJSON(conn, globals.OutgoingResponse{
		Type: "duplicate_file_response",
		Data: map[string]interface{}{"success": duplicatedFile.OriginalFileName},
	})
}
