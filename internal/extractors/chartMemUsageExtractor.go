package extractors

import (
	"fmt"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type ChartMemoryUsageExtractorOptions struct {
	Name         string
	Filename     string
	ChartOverlap []charts.Overlaper
}

func NewChartMemoryUsageExtractorOptions(processName string, filename string) ChartMemoryUsageExtractorOptions {
	return ChartMemoryUsageExtractorOptions{Name: processName, Filename: filename}
}

func (o *ChartMemoryUsageExtractorOptions) OverlapChart(c charts.Overlaper) {
	o.ChartOverlap = append(o.ChartOverlap, c)
}

type MemoryUsageChart struct {
	// ProcessName is the name of the process that the memory is referring to
	ProcessName string
	// Filename is the file to which it extracts the chart
	Filename string
	// Data is the memory usage data
	Data []MemoryUsageData
	// From is when the chart was created
	From time.Time
	// To is when the chart stopped watching for more data
	To time.Time
	// ChartsOverlap are the charts that overlap with this
	ChartsOverlap []charts.Overlaper
}

func NewChartMemoryExtractor(processName string, filename string, chartsOverlap []charts.Overlaper) *MemoryUsageChart {
	return &MemoryUsageChart{ProcessName: processName, Filename: filename, ChartsOverlap: chartsOverlap}
}

func (m *MemoryUsageChart) Add(data MemoryUsageData) error {
	if len(m.Data) == 0 {
		m.From = time.Now()
	}
	m.Data = append(m.Data, data)
	return nil
}

func (m *MemoryUsageChart) StopAndExtract() error {
	fs, err := os.Create(m.Filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", m.Filename, err)
	}
	defer fs.Close()

	m.To = time.Now()
	// create a new line instance
	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    fmt.Sprintf("Memory usage of %s", m.ProcessName),
			Subtitle: "The memory usage of the process",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider", Start: 0, End: 80}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
	)

	rssLine := m.GetRssLineData()
	vszLine := m.GetRssSwapLineData()
	timeParts := len(m.Data)
	timeXAxis := m.DivideTimeIntoParts(timeParts)

	// Put data into instance
	line.SetXAxis(timeXAxis).
		AddSeries("RSS", rssLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		AddSeries("RSS + Swap", vszLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
		)

	line.Overlap(m.ChartsOverlap...)

	line.Render(fs)

	m.Reset()

	fmt.Printf("html chart has been written at %s\n", m.Filename)

	return nil
}

func (m *MemoryUsageChart) Reset() {
	m.From = time.Now()
	m.To = time.Time{}
	m.Data = []MemoryUsageData{}
}

func (m *MemoryUsageChart) GetRssLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Data))
	for i := 0; i < len(m.Data); i++ {
		items[i] = opts.LineData{Value: m.Data[i].Rss / 1024}
	}
	return items
}

func (m *MemoryUsageChart) GetRssSwapLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Data))
	for i := 0; i < len(m.Data); i++ {
		items[i] = opts.LineData{Value: m.Data[i].RssSwap / 1024}
	}
	return items
}

// DivideTimeIntoParts returns a string formatted time that is divided into parts
func (m *MemoryUsageChart) DivideTimeIntoParts(parts int) []string {
	if parts == 0 {
		return nil
	}

	partsResult := []string{}

	totalTime := m.To.Sub(m.From)
	part := totalTime.Milliseconds() / int64(parts) // How much time each part has
	var t time.Time = m.From
	for i := 0; i < parts; i++ {
		t = t.Add(time.Millisecond * time.Duration(part))
		tStr := fmt.Sprintf("%2d:%2d:%2d", t.Hour(), t.Minute(), t.Second())
		partsResult = append(partsResult, tStr)
	}
	return partsResult
}
