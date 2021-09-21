package process

import "time"

type DarwinProcess struct {
	Pid int32
}

func NewDarwinProcess(pid int32) (*DarwinProcess, error) {
	return &DarwinProcess{Pid: pid}, nil
}

func (p *DarwinProcess) GetName() (string, error) {
	return "", nil
}
func (p *DarwinProcess) GetChildrenPids() ([]int32, error) {
	return nil, nil
}
func (p *DarwinProcess) GetStats() (ProcessStats, error) {
	return ProcessStats{}, nil
}
func (p *DarwinProcess) WatchStats(interval time.Duration) <-chan ProcessStats {
	ch := make(chan ProcessStats)
	return ch
}
func (p *DarwinProcess) GetCpuUsage() (CpuUsage, error) {
	return CpuUsage{}, nil
}
func (p *DarwinProcess) GetMemoryUsage() (MemoryUsage, error) {
	return MemoryUsage{}, nil
}
func (p *DarwinProcess) GetRss() (int64, error) {
	return 0, nil
}
func (p *DarwinProcess) GetRssWithSwap() (int64, error) {
	return 0, nil
}
