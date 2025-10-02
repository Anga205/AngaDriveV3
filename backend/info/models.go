package info

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
