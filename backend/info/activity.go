package info

import (
	"service/database"
	"sort"
	"time"
)

func GetLast7DaysCounts() ([]string, []int64) {
	timestamps := database.TimeStamps
	dates := make([]string, 0, 7)
	counts := make([]int64, 0, 7)
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Kolkata")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	for i := -6; i <= 0; i++ {
		day := today.AddDate(0, 0, i)
		dayStr := day.Format("Jan 2")
		dates = append(dates, dayStr)
		counts = append(counts, 0)
	}

	if len(timestamps) == 0 {
		return dates, counts
	}

	for i := -6; i <= 0; i++ {
		day := today.AddDate(0, 0, i)
		startOfDay := day.Unix()
		endOfDay := day.Add(24 * time.Hour).Unix()

		startIdx := sort.Search(len(timestamps), func(j int) bool {
			return timestamps[j] >= startOfDay
		})

		endIdx := sort.Search(len(timestamps), func(j int) bool {
			return timestamps[j] >= endOfDay
		})

		counts[i+6] = int64(endIdx - startIdx)
	}
	return dates, counts
}
