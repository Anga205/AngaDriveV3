package database

import "strings"

type Account struct {
	Token          string `gorm:"primaryKey" json:"token"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	HashedPassword string `json:"-"`
}

type Activity struct {
	Timestamps int64
}

func (Activity) TableName() string {
	// Override the default table name, which would be "activities", idk why gorm does this
	// but it does, so we need to override it
	// DO NOT REMOVE THIS FUNCTION
	return "activity"
}

type Collection struct {
	ID          string `gorm:"primaryKey"`
	Name        string
	Editors     string // Comma separated list of AccountToken's
	Size        int
	Collections string // Comma separated list of Collection ID's
	Files       string // Comma separated list of FileDirectory's
	Hidden      bool   `gorm:"default:false"`
}

func (s Collection) GetEditors() []string {
	editors := strings.Split(s.Editors, ",")
	for i := range editors {
		editors[i] = strings.TrimSpace(editors[i])
	}
	return editors
}

func (s Collection) GetCollections() []string {
	collections := strings.Split(s.Collections, ",")
	for i := range collections {
		collections[i] = strings.TrimSpace(collections[i])
	}
	return collections
}

func (s Collection) GetFiles() []string {
	files := strings.Split(s.Files, ",")
	for i := range files {
		files[i] = strings.TrimSpace(files[i])
	}
	return files
}

type FileData struct {
	OriginalFileName string `json:"original_file_name"`
	FileDirectory    string `gorm:"primaryKey" json:"file_directory"`
	AccountToken     string `json:"account_token"`
	FileSize         int    `json:"file_size"`
	Timestamp        int64  `json:"timestamp"`
	Md5sum           string `json:"-"`
}
