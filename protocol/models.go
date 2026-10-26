// Package protocol defines the wire contract shared by the websocket and HTTP
// transports, the broadcast hub, and the service layer. It contains only data
// types and pure helpers so it can be imported by every layer without creating
// cycles.
package protocol

import (
	"angadrive/accounts"
	"angadrive/database"
	"fmt"
)

// GraphData is a generic chart payload sent to the homepage.
type GraphData struct {
	XAxis       []string `json:"x_axis"`
	YAxis       []int64  `json:"y_axis"`
	Label       string   `json:"label"`
	BeginAtZero bool     `json:"begin_at_zero"`
}

// FileUpdate tells a client to add (Toggle=true) or remove (Toggle=false) a
// file from its view. Replace indicates an in-place update.
type FileUpdate struct {
	Toggle  bool `json:"toggle"`
	Replace bool `json:"replace"`
	File    database.FileData
}

// CollectionCardData is the summary representation of a collection.
type CollectionCardData struct {
	CollectionID   string `json:"id"`
	CollectionName string `json:"name"`
	Size           int64  `json:"size"`
	FileCount      int    `json:"file_count"`
	FolderCount    int    `json:"folder_count"`
	EditorCount    int    `json:"editor_count"`
	Timestamp      int64  `json:"timestamp"`
}

// CollectionCardUpdate tells a client to add/remove a collection card.
type CollectionCardUpdate struct {
	Toggle     bool               `json:"toggle"`
	Collection CollectionCardData `json:"collection"`
}

// AuthInfo carries the credentials supplied with a request.
type AuthInfo struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// GetToken resolves the request's credentials to an account token, verifying
// the password when email/password are supplied.
func (a AuthInfo) GetToken() (string, error) {
	if (a.Email == "" || a.Password == "") && a.Token == "" {
		return "", fmt.Errorf("no authentication information provided")
	}
	if a.Email != "" && a.Password != "" {
		if !accounts.Authenticate(a.Email, a.Password) {
			return "", fmt.Errorf("authentication failed")
		}
		account, err := database.FindUserByEmail(a.Email)
		if err != nil {
			return "", fmt.Errorf("failed to find user by email: %v", err)
		}
		return account.Token, nil
	}
	if a.Token != "" {
		return a.Token, nil
	}
	return "", fmt.Errorf("invalid authentication information provided")
}

type ConvertVideoRequest struct {
	FileDirectory string   `json:"file_directory"`
	Auth          AuthInfo `json:"auth"`
}

type ConvertImageRequest struct {
	FileDirectory string   `json:"file_directory"`
	TargetFormat  string   `json:"target_format"`
	Auth          AuthInfo `json:"auth"`
}

type DeleteFileRequest struct {
	FileDirectory string   `json:"file_directory"`
	Auth          AuthInfo `json:"auth"`
}

type RenameFileRequest struct {
	FileDirectory string   `json:"file_directory"`
	NewFileName   string   `json:"new_file_name"`
	Auth          AuthInfo `json:"auth"`
}

type DuplicateFileRequest struct {
	FileDirectory string   `json:"file_directory"`
	Auth          AuthInfo `json:"auth"`
}

type BulkDeleteRequest struct {
	FileDirectories []string `json:"file_directories"`
	Auth            AuthInfo `json:"auth"`
}

type BulkDuplicateRequest struct {
	FileDirectories []string `json:"file_directories"`
	Auth            AuthInfo `json:"auth"`
}

type BulkAddFilesToCollectionRequest struct {
	FileDirectories []string `json:"file_directories"`
	CollectionID    string   `json:"collection_id"`
	CollectionName  string   `json:"collection_name"`
	Auth            AuthInfo `json:"auth"`
}

type FileDeleteError struct {
	FileDirectory string `json:"file_directory"`
	Error         string `json:"error"`
}

type BulkDeleteResponse struct {
	Deleted []string          `json:"deleted"`
	Errors  []FileDeleteError `json:"errors"`
}

type BulkDuplicateResponse struct {
	Duplicated []string          `json:"duplicated"`
	Errors     []FileDeleteError `json:"errors"`
}

type BulkAddFilesToCollectionResponse struct {
	CollectionID string            `json:"collection_id"`
	Added        []string          `json:"added"`
	Errors       []FileDeleteError `json:"errors"`
}

type CreateCollectionRequest struct {
	CollectionName string   `json:"collection_name"`
	Auth           AuthInfo `json:"auth"`
}

type DeleteCollectionRequest struct {
	CollectionID string   `json:"collection_id"`
	Auth         AuthInfo `json:"auth"`
}

type GetCollectionRequest struct {
	CollectionID string   `json:"id"`
	Auth         AuthInfo `json:"auth"`
}

func (r GetCollectionRequest) GetCollectionID() string { return r.CollectionID }

type GetCollectionResponse struct {
	CollectionID   string               `json:"collection_id"`
	CollectionName string               `json:"collection_name"`
	IsOwner        bool                 `json:"is_owner"`
	Files          []database.FileData  `json:"files"`
	Folders        []CollectionCardData `json:"folders"`
}

type AddFolderToCollectionRequest struct {
	FolderID     string   `json:"folder_id"`
	CollectionID string   `json:"collection_id"`
	Auth         AuthInfo `json:"auth"`
}

func (r AddFolderToCollectionRequest) GetCollectionID() string { return r.CollectionID }

type RemoveFolderFromCollectionRequest AddFolderToCollectionRequest

type CreateFolderInCollectionRequest struct {
	CollectionID string   `json:"collection_id"`
	FolderName   string   `json:"folder_name"`
	Auth         AuthInfo `json:"auth"`
}

func (r CreateFolderInCollectionRequest) GetCollectionID() string { return r.CollectionID }

type AddFileToCollectionRequest struct {
	FileDirectory string   `json:"file_directory"`
	CollectionID  string   `json:"collection_id"`
	Auth          AuthInfo `json:"auth"`
}

func (r AddFileToCollectionRequest) GetCollectionID() string { return r.CollectionID }

type RemoveFileFromCollectionRequest AddFileToCollectionRequest

type ImportGithubRepoRequest struct {
	RepoURL string   `json:"repo_url"`
	Auth    AuthInfo `json:"auth"`
}

type RemoveAccountRequest accounts.DeleteUserRequest

// CollectionCard builds the summary card for a collection.
func CollectionCard(collection database.Collection) CollectionCardData {
	return CollectionCardData{
		CollectionID:   collection.ID,
		CollectionName: collection.Name,
		Size:           int64(collection.Size),
		FileCount:      len(collection.GetFiles()),
		FolderCount:    len(collection.GetCollections()),
		EditorCount:    len(collection.GetEditors()),
		Timestamp:      collection.Timestamp,
	}
}

// BuildCollectionResponse expands a collection into its full response payload
// (files + child folders) for the given user token.
func BuildCollectionResponse(collection database.Collection, userToken string) GetCollectionResponse {
	resp := GetCollectionResponse{
		CollectionID:   collection.ID,
		CollectionName: collection.Name,
		IsOwner:        collection.IsEditor(userToken),
		Files:          []database.FileData{},
		Folders:        []CollectionCardData{},
	}
	for _, fileID := range collection.GetFiles() {
		file, err := database.GetFile(fileID)
		if err != nil {
			continue
		}
		resp.Files = append(resp.Files, file)
	}
	for _, folderID := range collection.GetCollections() {
		folder, err := database.GetCollection(folderID)
		if err != nil {
			continue
		}
		resp.Folders = append(resp.Folders, CollectionCard(folder))
	}
	return resp
}
