package extractors

import "github.com/exapsy/peakben/internal/extractors/chart"

type MemoryUsageExtractor interface {
	Add(rss int64, rssswap int64) error
	StopAndExtract() error
}

type ChartMemoryDataExtractorOptions struct {
	Name     string
	Filename string
}

func NewChartMemoryDataExtractorOptions(processName string, filename string) ChartMemoryDataExtractorOptions {
	return ChartMemoryDataExtractorOptions{Name: processName, Filename: filename}
}

type CsvMemoryDataExtractorOptions struct {
}

type MemoryDataExtractors struct {
	extractors []MemoryUsageExtractor
}

func NewMemoryDataExtractors(opts ...interface{}) MemoryDataExtractors {
	extractors := MemoryDataExtractors{}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case ChartMemoryDataExtractorOptions:
			chartExtractor := chart.NewMemoryUsageChart(opt.Name, opt.Filename)
			extractors.extractors = append(extractors.extractors, chartExtractor)
		}
	}

	return extractors
}

func (m *MemoryDataExtractors) Add(rss int64, rssswap int64) error {
	for _, e := range m.extractors {
		if err := e.Add(rss, rssswap); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryDataExtractors) StopAndExtract() error {
	for _, e := range m.extractors {
		if err := e.StopAndExtract(); err != nil {
			return err
		}
	}
	return nil
}
