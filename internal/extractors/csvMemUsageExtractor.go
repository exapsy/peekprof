package extractors

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

type CsvMemoryUsageExtractorOptions struct {
	Filename string
}

func NewCsvMemoryUsageExtractorOptions(filename string) CsvMemoryUsageExtractorOptions {
	return CsvMemoryUsageExtractorOptions{Filename: filename}
}

type CsvMemoryUsage struct {
	Filename string
	Data     []MemoryUsageData
}

func NewCsvMemoryUsageExtractor(filename string) *CsvMemoryUsage {
	return &CsvMemoryUsage{Filename: filename}
}

func (c *CsvMemoryUsage) Add(data MemoryUsageData) error {
	c.Data = append(c.Data, data)
	return nil
}

func (c *CsvMemoryUsage) headers() []string {
	return []string{"timestamp", "rss kb", "rss+swap kb"}
}

func (c *CsvMemoryUsage) records() [][]string {
	records := make([][]string, len(c.Data))

	for i := 0; i < len(c.Data); i++ {
		r := []string{
			c.Data[i].Timestamp.Local().Format(time.RFC3339),
			fmt.Sprintf("%d", c.Data[i].Rss),
			fmt.Sprintf("%d", c.Data[i].RssSwap),
		}
		records[i] = r
	}

	return records
}

func (c *CsvMemoryUsage) StopAndExtract() error {
	f, err := os.Create(c.Filename)
	if err != nil {
		return fmt.Errorf("failed to create csv file")
	}
	defer f.Close()

	csvWriter := csv.NewWriter(f)

	records := [][]string{
		c.headers(),
	}
	records = append(records, c.records()...)

	err = csvWriter.WriteAll(records)
	if err != nil {
		return fmt.Errorf("failed to write records: %v", err)
	}

	fmt.Printf("csv has been written at %s\n", c.Filename)

	return nil
}
