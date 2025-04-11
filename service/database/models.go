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
	OriginalFileName string
	FileDirectory    string `gorm:"primaryKey"`
	AccountToken     string
	FileSize         int
	Timestamp        int64
	Cached           bool `gorm:"default:false"`
}
