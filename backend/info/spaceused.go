package info

import (
	"service/database"
	"time"
)

func BeginningOfToday() int64 {
	timeZone := "Asia/Kolkata"
	loc, _ := time.LoadLocation(timeZone)
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return startOfDay.Unix()
}

func PastXDays() [X]string {
	timeZone := "Asia/Kolkata"
	loc, _ := time.LoadLocation(timeZone)
	now := time.Now().In(loc)
	var days [X]string
	for i := 0; i < X; i++ {
		days[i] = now.AddDate(0, 0, -i).Format("Jan 2")
	}
	return days
}

func GetSpaceUsedGraph() ([X]string, [X]int64, error) {
	startOfToday := BeginningOfToday()

	days := PastXDays()
	fileInfo, _ := database.GetAllFileSizesAndTimes()

	// this is the per day file sizes array that this function will eventually return. Last element is today, first element is $x days ago
	var fileSizes [X]int64

	var dailyIncrease [X + 1]int64
	const daySeconds = 60 * 60 * 24

	for _, file := range fileInfo {
		if file.Time >= startOfToday {
			dailyIncrease[0] += file.Size
			continue
		}

		found := false
		for i := 1; i <= X-1; i++ {
			if file.Time >= startOfToday-int64(i)*daySeconds {
				dailyIncrease[i] += file.Size
				found = true
				break
			}
		}
		if !found {
			// Older than $x days ago
			dailyIncrease[X] += file.Size
		}
	}

	// Calculate the cumulative total size for each of the last $x days.
	// fileSizes[$x-1] will be the total size today.
	// fileSizes[0] will be the total size $x days ago.
	var cumulativeSize int64
	for _, size := range dailyIncrease {
		cumulativeSize += size
	}
	fileSizes[X-1] = cumulativeSize

	// Calculate totals for previous days by subtracting the daily increases.
	for i := 0; i < X-1; i++ {
		cumulativeSize -= dailyIncrease[i]
		fileSizes[X-2-i] = cumulativeSize
	}
	for i, j := 0, len(days)-1; i < j; i, j = i+1, j-1 {
		days[i], days[j] = days[j], days[i]
	}
	return days, fileSizes, nil
}
