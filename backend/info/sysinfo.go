package info

import (
	"log"
	"os"
	"path/filepath"
	"service/database"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func getCPUinfo() (CPUInfo, error) {
	percentage, err := cpu.Percent(500*time.Millisecond, false)
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

	totalRAM := v.Total
	if envRAM := os.Getenv("RAM_AVAILABLE"); envRAM != "" {
		if ramValue, err := strconv.ParseUint(envRAM, 10, 64); err == nil {
			totalRAM = ramValue
		}
	}

	ramInfo := RAMInfo{
		TotalRAM:       totalRAM,
		UsedRAM:        v.Used,
		AvailableRAM:   v.Available,
		RAMPercentUsed: v.UsedPercent,
	}
	return ramInfo, nil
}

func getSpaceUsed() (int, error) {
	var totalSize int
	err := filepath.Walk(database.UploadedFilesDir, func(_ string, info os.FileInfo, err error) error {
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

var currentSysInfo SystemInfo

func InitializeSysInfo() {
	go func() {
		for {
			cpuInfo, _ := getCPUinfo()
			currentSysInfo.CPU = cpuInfo
			time.Sleep(50 * time.Millisecond)
		}
	}()

	go func() {
		for {
			ramInfo, _ := getRAMinfo()
			currentSysInfo.RAM = ramInfo
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		for {
			spaceUsed, _ := getSpaceUsed()
			currentSysInfo.SpaceUsed = spaceUsed
			time.Sleep(1 * time.Minute)
		}
	}()
}

func GetSysInfo() (SystemInfo, error) {
	return currentSysInfo, nil
}
