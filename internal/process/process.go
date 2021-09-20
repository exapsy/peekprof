package process

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
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
	var statusFile *os.File
	for {
		s, err := os.Open(statusDir(pid))
		statusFile = s
		if err != nil {
			continue
		}
		break
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

type ProcessState string

const (
	ProcessStateSleeping            ProcessState = "S"
	ProcessStateRunning             ProcessState = "R"
	ProcessStateZombie              ProcessState = "Z"
	ProcessStateUninterruptibleWait ProcessState = "D"
)

type ProcessStatus struct {
	Name         string
	VmPeakMemory int64
	VmSize       int64
}

// IsRunning sends signal 0, which is a signal for nothing but still performs error checking
func (p *Process) IsRunning() (bool, error) {
	pid := p.Pid
	if pid <= 0 {
		return false, fmt.Errorf("invalid pid %v", pid)
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return false, err
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if err.Error() == "os: process already finished" {
		return false, nil
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false, err
	}
	switch errno {
	case syscall.ESRCH:
		return false, nil
	case syscall.EPERM:
		return true, nil
	}
	return false, err
}

func getStatus(pid int32) (*ProcessStatus, error) {
	sf, err := os.Open(statusDir(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to open status file: %w", err)
	}
	defer sf.Close()
	smap, err := readStatusMap(sf)

	if err != nil {
		return nil, fmt.Errorf("failed to load process status: %w", err)
	}

	vmSizeStr := strings.Split(strings.Trim(smap["VmSize"], "\n \t"), " ")[0]
	vmSize, err := strconv.ParseInt(vmSizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VmSize: %w", err)
	}

	pc := &ProcessStatus{
		Name:   smap["Name"],
		VmSize: vmSize,
	}

	return pc, nil
}

func (p *Process) GetStatus() (*ProcessStatus, error) {
	var smap map[string]string
	var err error
	for {
		smap, err = readStatusMap(p.statusFile)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load process status: %w", err)
	}

	pc := &ProcessStatus{
		Name: smap["Name"],
	}

	return pc, nil
}

func (p *Process) GetState() ProcessState {
	cmd := fmt.Sprintf("ps -q %d -o state --no-headers", p.Pid)
	e, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		panic("error getting process status")
	}
	eStr := strings.Trim(string(e), " \n")
	return ProcessState(eStr)
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
		if interval == 0 {
			interval = time.Second
		}
		tick := time.NewTicker(interval)
		defer close(ch)
		defer tick.Stop()
		for range tick.C {
			mu, err := p.GetMemoryUsage()
			if err != nil {
				log.Fatalf("failed to get memory usage: %s\n", err)
				continue
			}
			mus, err := p.GetMemoryUsageWithSwap()
			if err != nil {
				continue
			}
			if mus == 0 || mu == 0 {
				continue
			}
			ch <- MemUsage{Rss: mu, Vsz: mus}
		}
	}(ch)
	return ch
}

func (p *Process) GetChildrenPids() ([]int32, error) {
	cmd := strings.Fields(fmt.Sprintf("pgrep -P %d", p.Pid))
	pidsBytes, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		return nil, nil
	}
	pidsBytes = []byte(strings.Trim(string(pidsBytes), "\n "))
	pidsStrArr := strings.Split(string(pidsBytes), "\n")
	var pids []int32
	for _, pid := range pidsStrArr {
		pid = strings.Trim(pid, "\n ")
		p, err := strconv.Atoi(pid)
		if err != nil {
			return nil, fmt.Errorf("failec converting %q to int: %s", pid, err)
		}
		pids = append(pids, int32(p))
	}
	return pids, nil
}

// GetMemoryUsage returns the current memory usage in kilobytes of the process.
// This is calculated from the total RSS from all the libraries and itself
// that the process uses. RSS includes heap and stack memory, but not swap memory.
func (p *Process) GetMemoryUsage() (int64, error) {
	children, err := p.GetChildrenPids()
	children = append(children, p.Pid)
	if err != nil {
		return 0, err
	}
	var total int64 = 0
	for _, child := range children {
		cmd := fmt.Sprintf(`cat %s | grep -i rss |  awk '{Total+=$2} END {print Total}'`, smapsDir(child))
		rss, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
		}
		rss = []byte(strings.Trim(string(rss), "\n "))
		if len(rss) == 0 {
			continue
		}

		memUsage, err := strconv.Atoi(string(rss))
		if err != nil {
			return 0, fmt.Errorf("failed to convert output %q to int: %w", rss, err)
		}
		total = total + int64(memUsage)
	}

	return total, err
}

// GetMemoryUsageWithSwap returns the current memory usage in kilobytes of the process.
// This is calculated from the total memory from all the libraries and itself
// that the process uses.
func (p *Process) GetMemoryUsageWithSwap() (int64, error) {
	children, err := p.GetChildrenPids()
	children = append(children, p.Pid)
	if err != nil {
		return 0, err
	}
	var total int64 = 0
	for _, child := range children {
		cmd := fmt.Sprintf(`cat %s | grep -i swap |  awk '{Total+=$2} END {print Total}'`, smapsDir(child))
		rss, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
		}

		rss = []byte(strings.Trim(string(rss), "\n "))
		if len(rss) == 0 {
			continue
		}

		swapUsage, err := strconv.Atoi(string(rss))
		if err != nil {
			return 0, fmt.Errorf("failed to convert size to int: %w", err)
		}

		memUsage, err := p.GetMemoryUsage()
		if err != nil {
			return 0, fmt.Errorf("failed to get memory and swap usage: %w", err)
		}
		total = total + memUsage + int64(swapUsage)
	}

	return total, err
}

func (p *Process) GetName() (string, error) {
	status, err := p.GetStatus()
	if err != nil {
		return "", fmt.Errorf("could not get status: %w", err)
	}
	return status.Name, nil
}
