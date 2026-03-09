package info

// this is the number of days for which we want to track site activity and space used. It can be changed as needed, but make sure to change it in all places where it's used.
const X = 7

type RAMInfo struct {
	TotalRAM       uint64  `json:"total_ram"`
	UsedRAM        uint64  `json:"used_ram"`
	FreeRAM        uint64  `json:"free_ram"`
	RAMPercentUsed float64 `json:"ram_percent_used"`
}

type CPUInfo struct {
	CPUModelName string  `json:"cpu_model_name"`
	CPUUsage     float64 `json:"cpu_usage"`
}

type SystemInfo struct {
	CPU       CPUInfo `json:"cpu"`
	RAM       RAMInfo `json:"ram"`
	SpaceUsed int     `json:"space_used"`
}
