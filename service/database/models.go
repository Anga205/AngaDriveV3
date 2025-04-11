package database

type Account struct {
	Token          string `gorm:"primaryKey"`
	DisplayName    string
	Email          string
	HashedPassword string
}

type Activity struct {
	Timestamps int64
}

type Collection struct {
	ID          string `gorm:"primaryKey"`
	Name        string
	Editors     string
	Size        int
	Collections string
	Files       string
	Hidden      bool `gorm:"default:false"`
}

type FileData struct {
	OriginalFileName string `json:"original_file_name"`
	FileDirectory    string `gorm:"primaryKey" json:"file_directory"`
	AccountToken     string `json:"account_token"`
	FileSize         int    `json:"file_size"`
	Timestamp        int64  `json:"timestamp"`
	Cached           bool   `gorm:"default:false" json:"cached"`
}
