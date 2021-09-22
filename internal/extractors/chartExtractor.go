package extractors

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type ChartExtractorOptions struct {
	ProcessName            string
	Filename               string
	UpdateLiveListenWSHost string
}

func NewChartExtractorOptions(processname string, filename string) ChartExtractorOptions {
	return ChartExtractorOptions{
		ProcessName: processname,
		Filename:    filename,
	}
}

func (o *ChartExtractorOptions) UpdateLive(host string) *ChartExtractorOptions {
	o.UpdateLiveListenWSHost = host
	return o
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
	To                     time.Time
	UpdateLiveListenWSHost string
	file                   *os.File
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func NewChartExtractor(opts ChartExtractorOptions) *ChartExtractor {
	fs, err := os.Create(opts.Filename)
	if err != nil {
		panic(fmt.Errorf("failed to create file %s: %w", opts.Filename, err))
	}
	chartExtractor := &ChartExtractor{
		ProcessName:            opts.ProcessName,
		Filename:               opts.Filename,
		UpdateLiveListenWSHost: opts.UpdateLiveListenWSHost,
		file:                   fs,
	}

	// Generate html page for live updates
	if opts.UpdateLiveListenWSHost != "" {
		page := chartExtractor.generateChartsPage()
		page.Render(fs)
		fpath, err := filepath.Abs(opts.Filename)
		if err != nil {
			panic(fmt.Errorf("could not open file in browser: %w", err))
		}
		openBrowser(fpath)
	}

	return chartExtractor
}

func (m *ChartExtractor) Add(data ProcessStatsData) error {
	if len(m.Data) == 0 {
		m.From = time.Now()
	}
	m.Data = append(m.Data, data)

	return nil
}

func (m *ChartExtractor) StopAndExtract() error {
	defer m.file.Close()
	defer m.reset()

	m.To = time.Now()

	page := m.generateChartsPage()
	page.Render(m.file)

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

	if m.UpdateLiveListenWSHost != "" {
		m.AddCpuLineLiveUpdateJSFuncs(line)
	}

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
		line.AddSeries("RSS+Swap", rssSwapLine, charts.WithLabelOpts(opts.Label{Show: true, Position: "top"}))
	}

	if m.UpdateLiveListenWSHost != "" {
		m.AddMemoryLineLiveUpdateJSFuncs(line)
	}

	return line
}

func (m *ChartExtractor) AddMemoryLineLiveUpdateJSFuncs(line *charts.Line) {
	js := fmt.Sprintf(`{
	console.log("initializing memory event listener");
	const sseMem = new EventSource('http://%s/process/updates');

	sseMem.addEventListener("message", (e) => {
		const stats = JSON.parse(e.data);
		const rssData = [];
		const rssSwapData = [];
		const xAxisData = [];
		for (const stat of stats) {
			rssData.push({"value": stat.memoryUsage.rss, "XAxisIndex": 0, "YAxisIndex": 0});
			rssSwapData.push({"value": stat.memoryUsage.rssSwap, "XAxisIndex": 0, "YAxisIndex": 0});
			xAxisData.push(new Date(stat.timestamp).toISOString().slice(11, 20));
		}
		const option = {
			"dataZoom":[
				{
					"type":"slider",
					"end": 100
				}
			],
			"series":[
				{
					"name":"RSS",
					"data": rssData,
				},
				{
					"name":"RSS+Swap",
					"data": rssSwapData,
				}
			],
			"xAxis":[{"data":xAxisData}],
		};
		goecharts_%s.setOption(option);
	});
	}`, m.UpdateLiveListenWSHost, line.ChartID)

	line.AddJSFuncs(js)
}

func (m *ChartExtractor) AddCpuLineLiveUpdateJSFuncs(line *charts.Line) {
	js := fmt.Sprintf(`{
	console.log("initializing cpu event listener");
	const sseCpu = new EventSource('http://%s/process/updates');

	sseCpu.addEventListener("message", (e) => {
		const stats = JSON.parse(e.data);
		const data = [];
		const xAxisData = [];
		for (const stat of stats) {
			data.push({"value": stat.cpuUsage.percentage.toString(), "XAxisIndex": 0, "YAxisIndex": 0});
			xAxisData.push(new Date(stat.timestamp).toISOString().slice(11, 20));
		}
		const option = {
			"dataZoom":[
				{
					"type":"slider",
					"end": 100
				}
			],
			"series":[
				{
					"name":"CPU usage",
					"type":"line",
					"smooth":true,
					"waveAnimation":false,
					"renderLabelForZeroData":false,
					"selectedMode":false,
					"animation":true,
					"data": data,
				}
			],
			"xAxis":[{"data":xAxisData}],
		};
		goecharts_%s.setOption(option);
	});
	}`, m.UpdateLiveListenWSHost, line.ChartID)

	line.AddJSFuncs(js)
}

func (m *ChartExtractor) reset() {
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
		parts = 1
	}

	partsResult := []string{}

	totalTime := m.To.Sub(m.From)
	part := totalTime.Milliseconds() / int64(parts) // How much time each part has
	var t time.Time = m.From
	for i := 0; i < parts; i++ {
		t = t.Add(time.Millisecond * time.Duration(part))
		tStr := fmt.Sprintf(t.Format(time.RFC3339)[11:19])
		partsResult = append(partsResult, tStr)
	}
	return partsResult
}
