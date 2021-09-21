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
	"time"
)

type LinuxProcess struct {
	Pid        int32
	statusFile *os.File
}

const (
	LinuxProcStatusPath = "/proc/{pid}/status"
	LinuxProcSmapsPath  = "/proc/{pid}/smaps"
)

func NewLinuxProcess(pid int32) (*LinuxProcess, error) {
	statusFile, err := loadStatusFile(pid)
	if err != nil {
		return nil, err
	}

	return &LinuxProcess{pid, statusFile}, nil
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

type linuxProcessStatus struct {
	Name         string
	VmPeakMemory int64
	VmSize       int64
}

func getStatus(pid int32) (*linuxProcessStatus, error) {
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

	pc := &linuxProcessStatus{
		Name:   smap["Name"],
		VmSize: vmSize,
	}

	return pc, nil
}

func (p *LinuxProcess) getStatus() (*linuxProcessStatus, error) {
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

	pc := &linuxProcessStatus{
		Name: smap["Name"],
	}

	return pc, nil
}

func (p *LinuxProcess) WatchStats(interval time.Duration) <-chan ProcessStats {
	ch := make(chan ProcessStats)

	go func() {
		if interval == 0 {
			panic("refresh interval must be non-zero")
		}
		defer close(ch)

		tick := time.NewTicker(interval)
		defer tick.Stop()

		for range tick.C {
			stats, err := p.GetStats()
			if err != nil {
				log.Fatalf("error getting stats: %v", err)
			}

			ch <- stats
		}
	}()

	return ch
}

func (p *LinuxProcess) GetStats() (ProcessStats, error) {
	emptyps := ProcessStats{}

	memUsage, err := p.GetMemoryUsage()
	if err != nil {
		return emptyps, fmt.Errorf("failed getting memory usage: %w", err)
	}

	cpuUsage, err := p.GetCpuUsage()
	if err != nil {
		return emptyps, fmt.Errorf("failed getting cpu usage: %w", err)
	}

	return ProcessStats{
		MemoryUsage: memUsage,
		CpuUsage:    cpuUsage,
	}, nil
}

func (p *LinuxProcess) GetCpuUsage() (CpuUsage, error) {
	emptycpu := CpuUsage{}

	cmd := fmt.Sprintf(`ps -p %d -o %%cpu | awk 'FNR == 2 {gsub(/ /,""); print}'`, p.Pid)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return emptycpu, fmt.Errorf("failed to run command: %v", err)
	}

	if len(out) == 0 {
		return emptycpu, fmt.Errorf("output from cpu usage command is empty")
	}

	outStr := strings.Trim(string(out), " \n")

	cpuPercent64, err := strconv.ParseFloat(outStr, 32)
	if err != nil {
		return emptycpu, fmt.Errorf("failed to parse output to float: %w", err)
	}
	cpuPercent := float32(cpuPercent64)

	return CpuUsage{
		Percentage: cpuPercent,
	}, nil
}

func (p *LinuxProcess) GetMemoryUsage() (MemoryUsage, error) {
	emptymu := MemoryUsage{}

	rss, err := p.GetRss()
	if err != nil {
		return emptymu, fmt.Errorf("failed getting process rss: %w", err)
	}
	rssSwap, err := p.GetRssWithSwap()
	if err != nil {
		return emptymu, fmt.Errorf("failed getting process rss with swap: %w", err)
	}

	return MemoryUsage{
		Rss:     rss,
		RssSwap: rssSwap,
	}, nil
}

func (p *LinuxProcess) getChildrenPids() ([]int32, error) {
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

// GetRss returns the current memory usage in kilobytes of the process.
// This is calculated from the total RSS from all the libraries and itself
// that the process uses. RSS includes heap and stack memory, but not swap memory.
func (p *LinuxProcess) GetRss() (int64, error) {
	children, err := p.getChildrenPids()
	children = append(children, p.Pid)
	if err != nil {
		return 0, err
	}
	var total int64 = 0
	for _, child := range children {
		cmd := fmt.Sprintf(`cat %s | grep -i rss |  awk '{Total+=$2} END {print Total}'`, smapsDir(child))
		rss, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			if err.Error() != "signal: interrupt" {
				return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
			}
		}
		rss = []byte(strings.Trim(string(rss), "\n "))
		if len(rss) == 0 {
			continue
		}

		memUsage, err := strconv.ParseInt(string(rss), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to convert output %q to int: %w", rss, err)
		}
		total = total + memUsage
	}

	return total, err
}

// GetRssWithSwap returns the current memory usage in kilobytes of the process.
// This is calculated from the total memory from all the libraries and itself
// that the process uses.
func (p *LinuxProcess) GetRssWithSwap() (int64, error) {
	children, err := p.getChildrenPids()
	children = append(children, p.Pid)
	if err != nil {
		return 0, err
	}
	var total int64 = 0
	for _, child := range children {
		cmd := fmt.Sprintf(`cat %s | grep -i swap |  awk '{Total+=$2} END {print Total}'`, smapsDir(child))
		rss, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			if err.Error() != "signal: interrupt" {
				return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
			}
		}

		rss = []byte(strings.Trim(string(rss), "\n "))
		if len(rss) == 0 {
			continue
		}

		swapUsage, err := strconv.Atoi(string(rss))
		if err != nil {
			return 0, fmt.Errorf("failed to convert size to int: %w", err)
		}

		memUsage, err := p.GetRss()
		if err != nil {
			return 0, fmt.Errorf("failed to get memory and swap usage: %w", err)
		}
		total = total + memUsage + int64(swapUsage)
	}

	return total, err
}

func (p *LinuxProcess) GetName() (string, error) {
	status, err := p.getStatus()
	if err != nil {
		return "", fmt.Errorf("could not get status: %w", err)
	}
	return status.Name, nil
}
