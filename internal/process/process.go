package process

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Process struct {
	Pid        int32
	statusFile *os.File
}

const (
	LinuxProcStatusPath = "/proc/{pid}/status"
	LinuxProcSmapsPath  = "/proc/{pid}/smaps"
)

func NewProcess(pid int32) (*Process, error) {
	statusFile, err := loadStatusFile(pid)
	if err != nil {
		return nil, err
	}

	return &Process{pid, statusFile}, nil
}

func statusDir(pid int32) string {
	path := strings.Replace(LinuxProcStatusPath, "{pid}", fmt.Sprintf("%d", pid), 1)
	return path
}

func smapsDir(pid int32) string {
	path := strings.Replace(LinuxProcSmapsPath, "{pid}", fmt.Sprintf("%d", pid), 1)
	return path
}

func loadStatusFile(pid int32) (*os.File, error) {
	statusFile, err := os.Open(statusDir(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to open process status: %w", err)
	}

	return statusFile, nil
}

func readStatusMap(statusFile *os.File) (map[string]string, error) {
	b, err := ioutil.ReadAll(statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %w", err)
	}

	statusMap := map[string]string{}

	values := strings.Split(string(b), "\n")

	for _, v := range values {
		keyValue := strings.Split(v, ":\t")
		key := keyValue[0]
		if key == "" {
			break
		}
		value := keyValue[1]
		statusMap[key] = value
	}

	statusFile.Seek(0, io.SeekStart)

	return statusMap, nil
}

type ProcessStatus struct {
	VmPeakMemory int64
}

func (p *Process) GetStatus() (*ProcessStatus, error) {
	smap, err := readStatusMap(p.statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load process status: %w", err)
	}

	pc := &ProcessStatus{
		VmPeakMemory: parseStatusValueKb(smap["VmPeak"]),
	}

	return pc, nil
}

// readStatusValueKb returns the amounts of kb from a value
func parseStatusValueKb(value string) int64 {
	value = strings.Trim(value, " \t")
	value = strings.Replace(value, " kB", "", 1)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Errorf("failed parsing value %q to int64: %w", value, err))
	}

	return int64(parsed)
}

// GetPeakMemory returns the peak memory usage the process has reached.
func (p *Process) GetPeakMemory() (int64, error) {
	s, err := p.GetStatus()
	if err != nil {
		return 0, fmt.Errorf("failed to get process status: %w", err)
	}
	return s.VmPeakMemory, err
}

type MemUsage struct {
	Rss int64
	Vsz int64
}

func (p *Process) WatchMemoryUsage(interval time.Duration) <-chan MemUsage {
	ch := make(chan MemUsage)
	go func(ch chan MemUsage) {
		tick := time.NewTicker(interval)
		defer close(ch)
		defer tick.Stop()
		for range tick.C {
			mu, err := p.GetMemoryUsage()
			if mu == 0 {
				break
			}
			mus, err := p.GetMemoryUsageWithSwap()
			if mus == 0 {
				break
			}
			if err != nil {
				fmt.Printf("failed to get memory usage: %s\n", err)
				break
			}
			ch <- MemUsage{Rss: mu, Vsz: mus}
		}
	}(ch)
	return ch
}

// GetMemoryUsage returns the current memory usage in kilobytes of the process.
// This is calculated from the total RSS from all the libraries and itself
// that the process uses. RSS includes heap and stack memory, but not swap memory.
func (p *Process) GetMemoryUsage() (int64, error) {
	cmd := fmt.Sprintf(`cat %s | grep -i rss |  awk '{Total+=$2} END {print Total}'`, smapsDir(p.Pid))
	rss, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
	}

	memUsage, err := strconv.Atoi(strings.Trim(string(rss), "\n "))
	if err != nil {
		return 0, fmt.Errorf("failed to convert rss to int: %w", err)
	}

	return int64(memUsage), err
}

// GetMemoryUsageWithSwap returns the current memory usage in kilobytes of the process.
// This is calculated from the total memory from all the libraries and itself
// that the process uses.
func (p *Process) GetMemoryUsageWithSwap() (int64, error) {
	cmd := fmt.Sprintf(`cat %s | grep -i swap |  awk '{Total+=$2} END {print Total}'`, smapsDir(p.Pid))
	rss, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
	}

	swapUsage, err := strconv.Atoi(strings.Trim(string(rss), "\n "))
	if err != nil {
		return 0, fmt.Errorf("failed to convert size to int: %w", err)
	}

	memUsage, err := p.GetMemoryUsage()
	if err != nil {
		return 0, fmt.Errorf("failed to get memory usage: %w", err)
	}

	return memUsage + int64(swapUsage), err
}
