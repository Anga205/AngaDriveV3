package database

func PushTimeStamp(timestamp int64) {
	TimeStampsMutex.Lock()
	TimeStamps = append(TimeStamps, timestamp)
	TimeStampsMutex.Unlock()

	GetDB().Model(&Activity{}).Create(&Activity{Timestamps: timestamp})
}
