package process

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
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
	cmd := fmt.Sprintf("ps -p %d -c -o cmd | awk 'FNR == 2 {print}", p.Pid)
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
	}, nil
}

func (p *DarwinProcess) GetCpuUsage() (CpuUsage, error) {
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
func (p *DarwinProcess) GetMemoryUsage() (MemoryUsage, error) {
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
func (p *DarwinProcess) GetRss() (int64, error) {
	cmd := fmt.Sprintf("ps -p %d -o rss", p.Pid)
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		if err.Error() != "signal: interrupt" {
			return 0, fmt.Errorf("failed executing command %s: %s", cmd, err)
		}
	}
	output = []byte(strings.Trim(string(output), "\n "))
	if len(output) == 0 {
		return 0, nil
	}

	rss, err := strconv.ParseInt(string(output), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert output %q to int: %w", output, err)
	}

	return rss, nil
}
func (p *DarwinProcess) GetRssWithSwap() (int64, error) {
	return p.GetRss()
}
