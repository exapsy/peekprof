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
	// MemoryUsage throughout the whole time
	MemoryUsage []int64
	From        time.Time
	To          time.Time
}

func NewMemoryUsageChart() *MemoryUsageChart {
	return &MemoryUsageChart{}
}

func (m *MemoryUsageChart) AddValue(v int64) {
	if len(m.MemoryUsage) == 0 {
		m.From = time.Now()
	}
	m.MemoryUsage = append(m.MemoryUsage, v)
}

func (m *MemoryUsageChart) StopAndGenerateChart(w io.Writer) {
	m.To = time.Now()
	// create a new line instance
	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Memory usage",
			Subtitle: "The memory usage of the process",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider", Start: 0, End: 80}),
	)

	lineData := m.GetLineData()
	timeParts := len(m.MemoryUsage)
	timeXAxis := m.DivideTimeIntoParts(timeParts)

	// Put data into instance
	line.SetXAxis(timeXAxis).
		AddSeries("Memory Usage", lineData, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"})).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
		)
	line.Render(w)

	m.Reset()
}

func (m *MemoryUsageChart) Reset() {
	m.From = time.Now()
	m.To = time.Time{}
	m.MemoryUsage = []int64{}
}

func (m *MemoryUsageChart) GetLineData() []opts.LineData {
	items := make([]opts.LineData, len(m.MemoryUsage))
	for i := 0; i < len(m.MemoryUsage); i++ {
		items[i] = opts.LineData{Value: m.MemoryUsage[i] / 1024}
	}
	return items
}

// DivideTimeIntoParts returns a string formatted time that is divided into parts
func (m *MemoryUsageChart) DivideTimeIntoParts(parts int) []string {
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
