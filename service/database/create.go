package database

import (
	"fmt"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func createUploadedFilesDir(dirName string) error {
	err := os.Mkdir(dirName, 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("createUploadedFilesDir: %w", err)
	}
	return nil
}

func CreateDatabase(uploadedFilesDir string) error {
	err := createUploadedFilesDir(uploadedFilesDir)
	if err != nil {
		return fmt.Errorf("CreateDatabase: %w", err)
	}
	dbPath := uploadedFilesDir + string(os.PathSeparator) + "rx.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("CreateDatabase: %w", err)
	}

	err = db.AutoMigrate(&Account{}, &Activity{}, &Collection{}, &FileData{})
	if err != nil {
		return fmt.Errorf("CreateDatabase: %w", err)
	}
	fmt.Println("[GIN-debug] Database created successfully at", dbPath)
	return nil
}
