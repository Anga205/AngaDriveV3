package info

import (
	"service/database"
	"time"
)

func BeginningOfToday() (int64, error) {
	timeZone := "Asia/Kolkata"
	loc, _ := time.LoadLocation(timeZone)
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return startOfDay.Unix(), nil
}

func Past7Days() ([7]string, error) {
	timeZone := "Asia/Kolkata"
	loc, _ := time.LoadLocation(timeZone)
	now := time.Now().In(loc)
	var days [7]string
	for i := 0; i < 7; i++ {
		days[i] = now.AddDate(0, 0, -i).Format("Jan 2")
	}
	return days, nil
}

func GetSpaceUsedGraph() ([7]string, [7]int64, error) {
	startOfToday, err := BeginningOfToday()
	if err != nil {
		return [7]string{}, [7]int64{}, err
	}

	days, err := Past7Days()
	if err != nil {
		return [7]string{}, [7]int64{}, err
	}

	fileInfo, _ := database.GetAllFileSizesAndTimes()

	var fileSizes [7]int64

	var dailyIncrease [8]int64
	const daySeconds = 60 * 60 * 24

	for _, file := range fileInfo {
		if file.Time >= startOfToday {
			dailyIncrease[0] += file.Size
			continue
		}

		found := false
		for i := 1; i <= 6; i++ {
			if file.Time >= startOfToday-int64(i)*daySeconds {
				dailyIncrease[i] += file.Size
				found = true
				break
			}
		}
		if !found {
			// Older than 6 days ago
			dailyIncrease[7] += file.Size
		}
	}

	// Calculate the cumulative total size for each of the last 7 days.
	// fileSizes[6] will be the total size today.
	// fileSizes[0] will be the total size 6 days ago.
	var cumulativeSize int64
	for _, size := range dailyIncrease {
		cumulativeSize += size
	}
	fileSizes[6] = cumulativeSize

	// Calculate totals for previous days by subtracting the daily increases.
	for i := 0; i < 6; i++ {
		cumulativeSize -= dailyIncrease[i]
		fileSizes[5-i] = cumulativeSize
	}

	return days, fileSizes, nil
}
