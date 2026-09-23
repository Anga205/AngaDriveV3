package info

import (
	"angadrive/database"
	"sort"
	"time"
)

func GetLastXDaysCounts() ([]string, []int64) {
	timestamps := database.TimeStamps
	dates := make([]string, 0, DaysTracked)
	counts := make([]int64, 0, DaysTracked)
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	for i := -DaysTracked + 1; i <= 0; i++ {
		day := today.AddDate(0, 0, i)
		dayStr := day.Format("Jan 2")
		dates = append(dates, dayStr)
		counts = append(counts, 0)
	}

	if len(timestamps) == 0 {
		return dates, counts
	}

	for i := -DaysTracked + 1; i <= 0; i++ {
		day := today.AddDate(0, 0, i)
		startOfDay := day.Unix()
		endOfDay := day.Add(24 * time.Hour).Unix()

		startIdx := sort.Search(len(timestamps), func(j int) bool {
			return timestamps[j] >= startOfDay
		})

		endIdx := sort.Search(len(timestamps), func(j int) bool {
			return timestamps[j] >= endOfDay
		})

		counts[i+DaysTracked-1] = int64(endIdx - startIdx)
	}
	return dates, counts
}
