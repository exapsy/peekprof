package extractors

import "time"

type MemoryUsageData struct {
	Rss       int64
	RssSwap   int64
	Timestamp time.Time
}

type MemoryUsageExtractor interface {
	Add(data MemoryUsageData) error
	StopAndExtract() error
}

type MemoryUsageExtractors struct {
	extractors []MemoryUsageExtractor
}

func NewMemoryUsageExtractors(opts ...interface{}) MemoryUsageExtractors {
	extractors := MemoryUsageExtractors{}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case ChartMemoryUsageExtractorOptions:
			chartExtractor := NewChartMemoryExtractor(opt.Name, opt.Filename)
			extractors.extractors = append(extractors.extractors, chartExtractor)
		case CsvMemoryUsageExtractorOptions:
			csvExtractor := NewCsvMemoryUsageExtractor(opt.Filename)
			extractors.extractors = append(extractors.extractors, csvExtractor)
		}
	}

	return extractors
}

func (m *MemoryUsageExtractors) Add(d MemoryUsageData) error {
	for _, e := range m.extractors {
		if err := e.Add(d); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryUsageExtractors) StopAndExtract() error {
	for _, e := range m.extractors {
		if err := e.StopAndExtract(); err != nil {
			return err
		}
	}
	return nil
}
