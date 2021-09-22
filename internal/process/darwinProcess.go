package process

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type DarwinProcess struct {
	Pid int32
}

func NewDarwinProcess(pid int32) (*DarwinProcess, error) {
	return &DarwinProcess{Pid: pid}, nil
}

func (p *DarwinProcess) GetName() (string, error) {
	cmd := fmt.Sprintf("ps -p %d -c -o command | awk 'FNR == 2 {print}'", p.Pid)
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	outputStr := strings.TrimSpace(string(output))

	return outputStr, nil
}

func (p *DarwinProcess) WatchStats(interval time.Duration) <-chan ProcessStats {
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

func (p *DarwinProcess) GetStats() (ProcessStats, error) {
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
		Timestamp:   time.Now(),
	}, nil
}

func (p *DarwinProcess) GetCpuUsage() (CpuUsage, error) {
	emptycpu := CpuUsage{}

	cpuPercent64, err := getPsFloat(p.Pid, "%cpu")
	if err != nil {
		return emptycpu, fmt.Errorf("failed to get cpu value: %w", err)
	}
	cpuPercent := float32(cpuPercent64)

	return CpuUsage{
		Percentage: cpuPercent,
	}, nil
}

func (p *DarwinProcess) GetMemoryUsage() (MemoryUsage, error) {
	emptymu := MemoryUsage{}

	rss, err := p.GetRss()
	if err != nil {
		return emptymu, fmt.Errorf("failed getting process rss: %w", err)
	}

	return MemoryUsage{
		Rss:     rss,
		RssSwap: 0,
	}, nil
}

func (p *DarwinProcess) GetRss() (int64, error) {
	rss, err := getPsInt(p.Pid, "rss")
	if err != nil {
		return 0, err
	}

	return rss, nil
}
func (p *DarwinProcess) GetRssWithSwap() (int64, error) {
	return 0, fmt.Errorf("swap value is not supported for OSX")
}
