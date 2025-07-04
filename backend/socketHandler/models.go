package socketHandler

import (
	"service/database"
	"sync"

	"github.com/gorilla/websocket"
)

type UserInfo struct {
	Token          string
	Email          string
	HashedPassword string
}

type WebsocketData struct {
	Mutex           *sync.Mutex
	HomePageUpdates bool
	UserInfo        UserInfo
}

type IncomingMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type OutgoingResponse struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type GraphData struct {
	XAxis       []string `json:"x_axis"`
	YAxis       []int64  `json:"y_axis"`
	Label       string   `json:"label"`
	BeginAtZero bool     `json:"begin_at_zero"`
}

type FileUpdate struct {
	Toggle bool `json:"toggle"` // If this is true then it means "append a file to this page", false means the file already exists and we are removing it
	File   database.FileData
}

type CollectionCardData struct {
	CollectionID   string `json:"id"`
	CollectionName string `json:"name"`
	Size           int64  `json:"size"`
	FileCount      int    `json:"file_count"`
	FolderCount    int    `json:"folder_count"`
	EditorCount    int    `json:"editor_count"`
	Timestamp      int64  `json:"timestamp"`
}

type CollectionCardUpdate struct {
	Toggle     bool               `json:"toggle"` // same functionality as FileUpdate
	Collection CollectionCardData `json:"collection"`
}

type AuthInfo struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type ConvertVideoRequest struct {
	FileDirectory string   `json:"file_directory"`
	Auth          AuthInfo `json:"auth"`
}

type DeleteFileRequest struct {
	FileDirectory string   `json:"file_directory"`
	Auth          AuthInfo `json:"auth"`
}

type connInfo struct {
	conn *websocket.Conn
	data WebsocketData
}

type CreateCollectionRequest struct {
	CollectionName string   `json:"collection_name"`
	Auth           AuthInfo `json:"auth"`
}
