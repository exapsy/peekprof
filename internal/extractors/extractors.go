package extractors

import (
	"fmt"
	"time"
)

type MemoryUsageData struct {
	Rss     int64
	RssSwap int64
}

type CpuUsageData struct {
	Percentage float32
}

type ProcessStatsData struct {
	MemoryUsage MemoryUsageData
	CpuUsage    CpuUsageData
	Timestamp   time.Time
}

type Extractor interface {
	Add(data ProcessStatsData) error
	StopAndExtract() error
}

type Extractors struct {
	extractors []Extractor
}

func NewExtractors(opts ...interface{}) Extractors {
	extractors := Extractors{}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case ChartExtractorOptions:
			chartExtractor := NewChartExtractor(opt)
			extractors.extractors = append(extractors.extractors, chartExtractor)
		case CsvMemoryUsageExtractorOptions:
			csvExtractor, err := NewCsvMemoryUsageExtractor(opt.Filename)
			if err != nil {
				panic(fmt.Errorf("failed to create csv extractor: %w", err))
			}
			extractors.extractors = append(extractors.extractors, csvExtractor)
		}
	}

	return extractors
}

func (m *Extractors) Add(d ProcessStatsData) error {
	for _, e := range m.extractors {
		if err := e.Add(d); err != nil {
			return err
		}
	}
	return nil
}

func (m *Extractors) StopAndExtract() error {
	for _, e := range m.extractors {
		if err := e.StopAndExtract(); err != nil {
			return err
		}
	}
	return nil
}
