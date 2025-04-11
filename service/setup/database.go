package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"service/database"
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
		fmt.Printf("[GIN-debug] Database %s already exists\n", uploadedFilesDir)
		return nil
	} else {
		fmt.Printf("[GIN-debug] Database %s does not exist, creating...\n", uploadedFilesDir)
		err := database.CreateDatabase(uploadedFilesDir)
		if err != nil {
			return fmt.Errorf("SetupDB: %w", err)
		}
		return nil
	}
}
