package process

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type MemoryUsage struct {
	Rss     int64 `json:"rss"`
	RssSwap int64 `json:"rssSwap"`
}

type CpuUsage struct {
	Percentage float32 `json:"percentage"`
}

type ProcessStats struct {
	CpuUsage    CpuUsage    `json:"cpuUsage"`
	MemoryUsage MemoryUsage `json:"memoryUsage"`
	Timestamp   time.Time   `json:"timestamp"`
}

type Process interface {
	GetName() (string, error)
	GetStats() (ProcessStats, error)
	WatchStats(ctx context.Context, interval time.Duration) <-chan ProcessStats
	GetCpuUsage() (CpuUsage, error)
	GetMemoryUsage() (MemoryUsage, error)
	GetRss() (int64, error)
	GetSwap() (int64, error)
}

func NewProcess(pid int32) (Process, error) {
	switch runtime.GOOS {
	case "linux":
		process, err := NewLinuxProcess(pid)
		return process, err
	case "windows":
		panic("windows is not currently supported, yet")
		process, err := NewWindowsProcess(pid)
		return process, err
	case "darwin":
		process, err := NewDarwinProcess(pid)
		return process, err
	default:
		panic(fmt.Sprintf("%s is not currently supported, yet", runtime.GOOS))
	}
}

// getPsString is equivelant of running ps -p %pid -o %psKey.
// It's a command to get runtime information about a process.
// It is supported only in Unix like systems, like Linux, FreeBSD and OSX
func getPsString(pid int32, psKey string) (string, error) {
	cmd := fmt.Sprintf(`ps -p %d -o %s | awk 'FNR == 2 {gsub(/ /,""); print}'`, pid, psKey)
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		if err.Error() != "signal: interrupt" {
			return "", fmt.Errorf("failed executing command %s: %s", cmd, err)
		}
	}
	outputStr := strings.Trim(string(output), "\n ")
	if len(output) == 0 {
		return "", nil
	}

	return outputStr, nil
}

func getPsInt(pid int32, psKey string) (int64, error) {
	valueStr, err := getPsString(pid, psKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get ps string of type %s for pid %d: %w", psKey, pid, err)
	}

	value, err := strconv.ParseInt(string(valueStr), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert output %q to int: %w", valueStr, err)
	}

	return value, nil
}

func getPsFloat(pid int32, psKey string) (float64, error) {
	valueStr, err := getPsString(pid, psKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get ps string of type %s for pid %d: %w", psKey, pid, err)
	}

	value, err := strconv.ParseFloat(string(valueStr), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert output %q to float: %w", valueStr, err)
	}

	return value, nil
}
