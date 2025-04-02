package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

func DBexists(uploadedFilesDir string) bool {
	filePath := filepath.Join(uploadedFilesDir, "rx.db")

	if _, err := os.Stat(filePath); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		fmt.Printf("Error checking file: %v\n", err)
		return false
	}
}

func SetupDB(uploadedFilesDir string) error {
	if DBexists(uploadedFilesDir) {
		fmt.Printf("database %s already exists\n", uploadedFilesDir)
		return nil
	} else {
		fmt.Printf("Database %s does not exist, creating...\n", uploadedFilesDir)
		return nil
	}
}
