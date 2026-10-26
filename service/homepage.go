package service

import (
	"angadrive/hub"
	"angadrive/info"
	"angadrive/protocol"
)

// HomepageSnapshot is the initial set of homepage payloads sent when a client
// enables homepage updates.
type HomepageSnapshot struct {
	SiteActivity protocol.GraphData
	SpaceUsed    protocol.GraphData
	UserCount    int64
	FileCount    int64
	SysInfo      interface{}
}

// BuildHomepageSnapshot gathers the current homepage statistics.
func BuildHomepageSnapshot() HomepageSnapshot {
	xAxis, yAxis := info.GetLastXDaysCounts()
	days, sizes := hub.SpaceUsedSnapshot()
	sysInfo, _ := info.GetSysInfo()
	return HomepageSnapshot{
		SiteActivity: protocol.GraphData{XAxis: xAxis, YAxis: yAxis, Label: "Site Activity", BeginAtZero: true},
		SpaceUsed:    protocol.GraphData{XAxis: days, YAxis: sizes, Label: "Space Used", BeginAtZero: false},
		UserCount:    hub.UserCount(),
		FileCount:    hub.FileCount(),
		SysInfo:      sysInfo,
	}
}
