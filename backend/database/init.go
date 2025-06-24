package database

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func createUploadedFilesDir(dirName string) error {
	UploadedFilesDir = dirName
	if _, err := os.Stat(dirName); !os.IsNotExist(err) {
		return nil
	}

	err := os.Mkdir(dirName, 0755)
	if err != nil {
		return fmt.Errorf("createUploadedFilesDir: %w", err)
	}

	subDir := dirName + string(os.PathSeparator) + "i"
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		return fmt.Errorf("createUploadedFilesDir: %w", err)
	}
	subDir = dirName + string(os.PathSeparator) + "tmp_chunks"
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		return fmt.Errorf("createUploadedFilesDir: %w", err)
	}
	subDir = dirName + string(os.PathSeparator) + "pdf_previews"
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		return fmt.Errorf("createUploadedFilesDir: %w", err)
	}
	return nil
}

var (
	dbInstance       *gorm.DB
	UploadedFilesDir string
)

func InitializeDatabase(uploadedFilesDir string) error {
	err := createUploadedFilesDir(uploadedFilesDir)
	if err != nil {
		return fmt.Errorf("InitializeDatabase: %w", err)
	}
	dbPath := uploadedFilesDir + string(os.PathSeparator) + "rx.db"
	dbInstance, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("InitializeDatabase: %w", err)
	}

	err = dbInstance.AutoMigrate(&Account{}, &Activity{}, &Collection{}, &FileData{})
	if err != nil {
		return fmt.Errorf("InitializeDatabase: %w", err)
	}
	if gin.Mode() != gin.ReleaseMode {
		LoadCache()
	}
	fmt.Println("[GIN-debug] Database initialized successfully")
	return nil
}

func GetDB() *gorm.DB {
	if dbInstance == nil {
		panic("Database not initialized")
	}
	return dbInstance
}
