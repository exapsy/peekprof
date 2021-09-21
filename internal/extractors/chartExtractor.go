package extractors

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type ChartExtractorOptions struct {
	Name         string
	Filename     string
	ChartOverlap []charts.Overlaper
}

func NewChartExtractorOptions(processName string, filename string) ChartExtractorOptions {
	return ChartExtractorOptions{Name: processName, Filename: filename}
}

func (o *ChartExtractorOptions) OverlapChart(c charts.Overlaper) {
	o.ChartOverlap = append(o.ChartOverlap, c)
}

type ChartExtractor struct {
	// ProcessName is the name of the process that the memory is referring to
	ProcessName string
	// Filename is the file to which it extracts the chart
	Filename string
	// Data is the memory usage data
	Data []ProcessStatsData
	// From is when the chart was created
	From time.Time
	// To is when the chart stopped watching for more data
	To time.Time
	// ChartsOverlap are the charts that overlap with this
	ChartsOverlap []charts.Overlaper
}

func NewChartExtractor(processName string, filename string, chartsOverlap []charts.Overlaper) *ChartExtractor {
	return &ChartExtractor{ProcessName: processName, Filename: filename, ChartsOverlap: chartsOverlap}
}

func (m *ChartExtractor) Add(data ProcessStatsData) error {
	if len(m.Data) == 0 {
		m.From = time.Now()
	}
	m.Data = append(m.Data, data)
	return nil
}

func (m *ChartExtractor) StopAndExtract() error {
	fs, err := os.Create(m.Filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", m.Filename, err)
	}
	defer fs.Close()

	m.To = time.Now()

	page := m.generateChartsPage()
	page.Render(fs)

	m.Reset()

	fmt.Printf("html chart has been written at %s\n", m.Filename)

	return nil
}

func (m *ChartExtractor) generateChartsPage() *components.Page {
	memoryUsageChart := m.generateMemoryUsageChart()
	cpuUsageChart := m.generateCpuUsageChart()

	page := components.NewPage()
	page.AddCharts(
		memoryUsageChart,
		cpuUsageChart,
	)

	return page
}

func (m *ChartExtractor) generateCpuUsageChart() *charts.Line {
	// create a new line instance
	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    fmt.Sprintf("CPU usage of %s", m.ProcessName),
			Subtitle: "The cpu usage of the process",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider", Start: 0, End: 80}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
	)

	cpuPercentageLine := m.getCpuPercentageLineData()
	timeParts := len(m.Data)
	timeXAxis := m.DivideTimeIntoParts(timeParts)

	// Put data into instance
	line.SetXAxis(timeXAxis).
		AddSeries("CPU usage", cpuPercentageLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
		)

	return line
}

func (m *ChartExtractor) generateMemoryUsageChart() *charts.Line {
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

	rssLine := m.getRssLineData()
	timeParts := len(m.Data)
	timeXAxis := m.DivideTimeIntoParts(timeParts)

	// Put data into instance
	line.SetXAxis(timeXAxis).
		AddSeries("RSS", rssLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
		)

	if runtime.GOOS != "darwin" {
		rssSwapLine := m.getRssSwapLineData()
		line.AddSeries("RSS + Swap", rssSwapLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"}))
	}

	return line
}

func (m *ChartExtractor) Reset() {
	m.From = time.Now()
	m.To = time.Time{}
	m.Data = []ProcessStatsData{}
}

func (m *ChartExtractor) getCpuPercentageLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Data))
	for i := 0; i < len(m.Data); i++ {
		items[i] = opts.LineData{Value: fmt.Sprintf("%.1f", m.Data[i].CpuUsage.Percentage)}
	}
	return items
}

func (m *ChartExtractor) getRssLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Data))
	for i := 0; i < len(m.Data); i++ {
		items[i] = opts.LineData{Value: m.Data[i].MemoryUsage.Rss / 1024}
	}
	return items
}

func (m *ChartExtractor) getRssSwapLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Data))
	for i := 0; i < len(m.Data); i++ {
		items[i] = opts.LineData{Value: m.Data[i].MemoryUsage.RssSwap / 1024}
	}
	return items
}

// DivideTimeIntoParts returns a string formatted time that is divided into parts
func (m *ChartExtractor) DivideTimeIntoParts(parts int) []string {
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
