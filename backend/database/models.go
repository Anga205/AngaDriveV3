package database

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
