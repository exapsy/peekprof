package extractors

import (
	"encoding/csv"
	"fmt"
	"os"
	"runtime"
	"time"
)

type CsvMemoryUsageExtractorOptions struct {
	Filename string
}

func NewCsvExtractorOptions(filename string) CsvMemoryUsageExtractorOptions {
	return CsvMemoryUsageExtractorOptions{Filename: filename}
}

type CsvMemoryUsage struct {
	Filename  string
	Data      []ProcessStatsData
	file      *os.File
	csvWriter *csv.Writer
}

func NewCsvMemoryUsageExtractor(filename string) (*CsvMemoryUsage, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create csv file: %w", err)
	}

	csvWriter := csv.NewWriter(f)
	csvExtractor := &CsvMemoryUsage{Filename: filename, file: f, csvWriter: csvWriter}

	csvWriter.Write(csvExtractor.headers())

	return csvExtractor, nil
}

func (c *CsvMemoryUsage) Add(data ProcessStatsData) error {
	c.Data = append(c.Data, data)
	c.csvWriter.Write(c.dataToCsvRecord(data))
	return nil
}

func (c *CsvMemoryUsage) dataToCsvRecord(data ProcessStatsData) []string {
	var r []string

	timestamp := data.Timestamp.Local().Format(time.RFC3339)
	rss := fmt.Sprintf("%d", data.MemoryUsage.Rss)
	rssSwap := fmt.Sprintf("%d", data.MemoryUsage.RssSwap)
	cpuPercent := fmt.Sprintf("%.1f", data.CpuUsage.Percentage)

	if runtime.GOOS != "darwin" {
		r = []string{timestamp, rss, rssSwap, cpuPercent}
	} else {
		r = []string{timestamp, rss, cpuPercent}
	}

	return r
}

func (c *CsvMemoryUsage) headers() []string {
	var headers []string
	if runtime.GOOS != "darwin" {
		headers = []string{"timestamp", "rss kb", "rss+swap kb", "cpu%"}
	} else {
		headers = []string{"timestamp", "rss kb", "cpu%"}
	}
	return headers
}

func (c *CsvMemoryUsage) StopAndExtract() error {
	fmt.Printf("csv has been written at %s\n", c.Filename)
	c.csvWriter.Flush()
	err := c.file.Close()
	if err != nil {
		return fmt.Errorf("failed to close csv file")
	}

	return nil
}
