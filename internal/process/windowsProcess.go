package process

import (
	"context"
	"time"
)

type WindowsProcess struct {
	Pid int32
}

func NewWindowsProcess(pid int32) (*WindowsProcess, error) {
	return &WindowsProcess{Pid: pid}, nil
}

func (p *WindowsProcess) GetName() (string, error) {
	return "", nil
}
func (p *WindowsProcess) GetChildrenPids() ([]int32, error) {
	return nil, nil
}
func (p *WindowsProcess) GetStats() (ProcessStats, error) {
	return ProcessStats{}, nil
}
func (p *WindowsProcess) WatchStats(ctx context.Context, interval time.Duration) <-chan ProcessStats {
	ch := make(chan ProcessStats)
	return ch
}
func (p *WindowsProcess) GetCpuUsage() (CpuUsage, error) {
	return CpuUsage{}, nil
}
func (p *WindowsProcess) GetMemoryUsage() (MemoryUsage, error) {
	return MemoryUsage{}, nil
}
func (p *WindowsProcess) GetRss() (int64, error) {
	return 0, nil
}
func (p *WindowsProcess) GetSwap() (int64, error) {
	return 0, nil
}
