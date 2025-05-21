package database

import "fmt"

func DeleteFile(file FileData) error {

	go func() { // Remove from UserFiles Cache
		UserFilesMutex.Lock()
		for _, fileSet := range UserFiles {
			if fileSet != nil {
				fileSet.Remove(file)
			}
		}
		UserFilesMutex.Unlock()
	}()

	go func() { // Remove from CollectionFiles Cache
		CollectionFilesMutex.Lock()
		for _, fileSet := range CollectionFiles {
			if fileSet != nil {
				fileSet.Remove(file)
			}
		}
		CollectionFilesMutex.Unlock()
	}()

	// Delete from database
	db := GetDB()
	result := db.Delete(&FileData{}, file)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("file not found: %v", file)
	}
	return nil
}
