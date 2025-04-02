package sysinfo

import (
	"log"
	"os"
	"path/filepath"
	"service/global"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func getCPUinfo() (CPUInfo, error) {
	percentage, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		log.Fatal(err)
	}
	info, err := cpu.Info()
	if err != nil {
		log.Fatal(err)
	}

	cpuInfo := CPUInfo{
		CPUModelName: info[0].ModelName,
		CPUUsage:     percentage[0],
	}
	return cpuInfo, nil
}
func getRAMinfo() (RAMInfo, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Fatal(err)
	}
	ramInfo := RAMInfo{
		TotalRAM:       v.Total,
		UsedRAM:        v.Used,
		AvailableRAM:   v.Available,
		RAMPercentUsed: v.UsedPercent,
	}
	return ramInfo, nil
}

func getSpaceUsed() (int, error) {
	var totalSize int
	err := filepath.Walk(global.UploadedFilesDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += int(info.Size())
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalSize, nil
}

func GetSysInfo() (SystemInfo, error) {
	cpuInfo, err := getCPUinfo()
	if err != nil {
		return SystemInfo{}, err
	}

	ramInfo, err := getRAMinfo()
	if err != nil {
		return SystemInfo{}, err
	}

	spaceUsed, err := getSpaceUsed()
	if err != nil {
		spaceUsed = 0
	}

	sysInfo := SystemInfo{
		CPU:       cpuInfo,
		RAM:       ramInfo,
		SpaceUsed: spaceUsed,
	}

	return sysInfo, nil
}
