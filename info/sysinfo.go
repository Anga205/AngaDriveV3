package info

import (
	"angadrive/storage"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func getCPUinfo() (CPUInfo, error) {
	percentage, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return CPUInfo{}, err
	}
	info, err := cpu.Info()
	if err != nil {
		return CPUInfo{}, err
	}
	if len(info) == 0 || len(percentage) == 0 {
		return CPUInfo{}, nil
	}
	return CPUInfo{CPUModelName: info[0].ModelName, CPUUsage: percentage[0]}, nil
}

func getRAMinfo() (RAMInfo, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return RAMInfo{}, err
	}

	totalRAM := v.Total
	freeRAM := v.Available
	usePercentage := v.UsedPercent
	if envRAM := os.Getenv("RAM_AVAILABLE"); envRAM != "" {
		if ramValue, err := strconv.ParseUint(envRAM, 10, 64); err == nil {
			totalRAM = ramValue
			freeRAM = max(totalRAM-v.Used, 0)
			usePercentage = min((float64(v.Used)/float64(totalRAM))*100, 100)
		}
	}

	return RAMInfo{
		TotalRAM:       totalRAM,
		UsedRAM:        v.Used,
		FreeRAM:        freeRAM,
		RAMPercentUsed: usePercentage,
	}, nil
}

func getSpaceUsed() (int, error) {
	var totalSize int
	err := filepath.Walk(storage.Root(), func(_ string, info os.FileInfo, err error) error {
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

var (
	currentSysInfo SystemInfo
	sysInfoMu      sync.RWMutex
)

// InitializeSysInfo starts the background pollers that keep currentSysInfo up
// to date. Transient read errors are ignored rather than terminating the
// process.
func InitializeSysInfo() {
	go func() {
		for {
			if cpuInfo, err := getCPUinfo(); err == nil {
				sysInfoMu.Lock()
				currentSysInfo.CPU = cpuInfo
				sysInfoMu.Unlock()
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	go func() {
		for {
			if ramInfo, err := getRAMinfo(); err == nil {
				sysInfoMu.Lock()
				currentSysInfo.RAM = ramInfo
				sysInfoMu.Unlock()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		for {
			if spaceUsed, err := getSpaceUsed(); err == nil {
				sysInfoMu.Lock()
				currentSysInfo.SpaceUsed = spaceUsed
				sysInfoMu.Unlock()
			}
			time.Sleep(1 * time.Minute)
		}
	}()
}

// GetSysInfo returns a snapshot of the current system information.
func GetSysInfo() (SystemInfo, error) {
	sysInfoMu.RLock()
	defer sysInfoMu.RUnlock()
	return currentSysInfo, nil
}
