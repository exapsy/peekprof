package chart

import (
	"fmt"
	"io"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type MemoryUsageChart struct {
	// ProcessName is the name of the process that the memory is referring to
	ProcessName string
	// Rss is the total memory usage of the process without swap
	Rss []int64
	// Vsz is rss+swap
	Vsz  []int64
	From time.Time
	To   time.Time
}

func NewMemoryUsageChart(processName string) *MemoryUsageChart {
	return &MemoryUsageChart{ProcessName: processName}
}

func (m *MemoryUsageChart) AddValues(rss int64, vsz int64) {
	if len(m.Rss) == 0 {
		m.From = time.Now()
	}
	m.Rss = append(m.Rss, rss)
	m.Vsz = append(m.Vsz, vsz)
}

func (m *MemoryUsageChart) StopAndGenerateChart(w io.Writer) {
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
	vszLine := m.GetVszLineData()
	timeParts := len(m.Rss)
	timeXAxis := m.DivideTimeIntoParts(timeParts)

	// Put data into instance
	line.SetXAxis(timeXAxis).
		AddSeries("RSS Usage", rssLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		AddSeries("RSS + Swap", vszLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
		)
	line.Render(w)

	m.Reset()
}

func (m *MemoryUsageChart) Reset() {
	m.From = time.Now()
	m.To = time.Time{}
	m.Rss = []int64{}
	m.Vsz = []int64{}
}

func (m *MemoryUsageChart) GetRssLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Rss))
	for i := 0; i < len(m.Rss); i++ {
		items[i] = opts.LineData{Value: m.Rss[i] / 1024}
	}
	return items
}

func (m *MemoryUsageChart) GetVszLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.Vsz))
	for i := 0; i < len(m.Vsz); i++ {
		items[i] = opts.LineData{Value: m.Vsz[i] / 1024}
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
	fmt.Println(totalTime)
	part := totalTime.Milliseconds() / int64(parts) // How much time each part has
	var t time.Time = m.From
	for i := 0; i < parts; i++ {
		t = t.Add(time.Millisecond * time.Duration(part))
		tStr := fmt.Sprintf("%2d:%2d:%2d", t.Hour(), t.Minute(), t.Second())
		partsResult = append(partsResult, tStr)
	}
	return partsResult
}