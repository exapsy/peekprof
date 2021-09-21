package process

import (
	"runtime"
	"time"
)

type MemoryUsage struct {
	Rss     int64
	RssSwap int64
}

type CpuUsage struct {
	Percentage float32
}

type ProcessStats struct {
	CpuUsage    CpuUsage
	MemoryUsage MemoryUsage
}

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
		panic("windows is not currently yet supported")
		process, err := NewWindowsProcess(pid)
		return process, err
	case "darwin":
		panic("osx is not currently yet supported")
		process, err := NewDarwinProcess(pid)
		return process, err
	}
	return nil, nil
}
