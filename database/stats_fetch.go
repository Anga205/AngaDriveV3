package database

func GetCumulativeUserCount() (int64, error) {
	db := GetDB()
	var count int64
	err := db.Model(&Account{}).Distinct("token").Count(&count).Error
	if err != nil {
		return 0, err
	}

	var fileCount int64
	err = db.Model(&FileData{}).
		Distinct("account_token").
		Where("account_token NOT IN (SELECT token FROM accounts)").
		Count(&fileCount).Error
	if err != nil {
		return 0, err
	}

	var collectionCount int64
	// Note: this is technically wrong since editors is a comma-separated string
	// that for now just so happens to only contain one value.
	err = db.Model(&Collection{}).
		Distinct("editors").
		Where("editors NOT IN (SELECT token FROM accounts)").
		Where("editors NOT IN (SELECT account_token FROM file_data)").
		Count(&collectionCount).Error
	if err != nil {
		return 0, err
	}
	return count + fileCount + collectionCount, nil
}

func CountFiles() (int64, error) {
	db := GetDB()
	var count int64
	err := db.Model(&FileData{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAllFiles returns every file in the database.
func GetAllFiles() ([]FileData, error) {
	db := GetDB()
	var files []FileData
	err := db.Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

type SizeAndTime struct {
	Size int64
	Time int64
}

func GetAllFileSizesAndTimes() ([]SizeAndTime, error) {
	db := GetDB()
	var files []FileData
	err := db.Find(&files).Error
	if err != nil {
		return nil, err
	}

	var fileSizesAndTimes []SizeAndTime
	for _, file := range files {
		fileSizesAndTimes = append(fileSizesAndTimes, SizeAndTime{
			Size: file.FileSize,
			Time: file.Timestamp,
		})
	}
	return fileSizesAndTimes, nil
}
