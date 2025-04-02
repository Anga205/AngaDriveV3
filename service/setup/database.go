package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

func DBexists(name string) bool {
	uploadedFilesDir := "uploaded_files"
	filePath := filepath.Join(uploadedFilesDir, name)

	if _, err := os.Stat(filePath); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		fmt.Printf("Error checking file: %v\n", err)
		return false
	}
}

func SetupDB(name string) error {
	if DBexists(name) {
		fmt.Printf("database %s already exists\n", name)
		return nil
	} else {
		fmt.Printf("Database %s does not exist, creating...\n", name)
		return nil
	}
}
