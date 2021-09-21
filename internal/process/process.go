package process

import (
	"runtime"
	"time"
)

type Process interface {
	GetName() (string, error)
	GetChildrenPids() ([]int32, error)
	GetStats() (ProcessStats, error)
	WatchStats(interval time.Duration) <-chan ProcessStats
	GetCpuUsage() (CpuUsage, error)
	GetMemoryUsage() (MemoryUsage, error)
	GetRss() (int64, error)
	GetRssWithSwap() (int64, error)
}

func NewProcess(pid int32) (Process, error) {
	switch runtime.GOOS {
	case "linux":
		process, err := NewLinuxProcess(pid)
		return process, err
	case "windows":
	case "darwin":
	}
	return nil, nil
}
