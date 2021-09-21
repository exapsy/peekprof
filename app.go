package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/exapsy/peakben/internal/extractors"
	"github.com/exapsy/peakben/internal/process"
)

type App struct {
	process         *process.Process
	runsExecutable  bool
	executable      *exec.Cmd
	ctx             context.Context
	cancel          context.CancelFunc
	peakMem         int64
	htmlFilename    string
	csvFilename     string
	refreshInterval time.Duration
	memExtractor    extractors.MemoryUsageExtractors
}

type AppOptions struct {
	PID             int32
	RunsExecutable  bool
	Cmd             *exec.Cmd
	HtmlFilename    string
	CsvFilename     string
	RefreshInterval string
}

func NewApp(opts *AppOptions) *App {
	refreshInterval := parseStringToDuration(opts.RefreshInterval)

	p, err := process.NewProcess(opts.PID)
	if err != nil {
		panic(fmt.Sprintf("failed to get process: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	pname, err := p.GetName()
	if err != nil {
		panic(fmt.Errorf("could not get process name: %w", err))
	}

	memExtractors := []interface{}{}
	if opts.CsvFilename != "" {
		csvExtractorOpts := extractors.NewCsvMemoryUsageExtractorOptions(opts.CsvFilename)
		memExtractors = append(memExtractors, csvExtractorOpts)
	}
	if opts.HtmlFilename != "" {
		chartExtractorOpts := extractors.NewChartMemoryUsageExtractorOptions(pname, opts.HtmlFilename)
		memExtractors = append(memExtractors, chartExtractorOpts)
	}

	memExtractor := extractors.NewMemoryUsageExtractors(memExtractors...)

	return &App{
		runsExecutable:  opts.RunsExecutable,
		process:         p,
		ctx:             ctx,
		cancel:          cancel,
		executable:      opts.Cmd,
		peakMem:         0,
		htmlFilename:    opts.HtmlFilename,
		csvFilename:     opts.CsvFilename,
		refreshInterval: refreshInterval,
		memExtractor:    memExtractor,
	}
}

// parseStringToDuration parses a string of format <amount><unit> to time.Duration
// Example:
// 2s becomes 2 * time.Second
func parseStringToDuration(s string) time.Duration {
	reg := regexp.MustCompile(`(\d+)(\w+)`)
	arr := reg.FindStringSubmatch(s)
	if len(arr) != 3 {
		panic(fmt.Errorf("time %s is not of correct format <amount><unit> (unit: [s: seconds, m: minutes])", s))
	}
	amountStr := arr[1]
	unit := arr[2]
	var amount int
	var unitDur time.Duration

	switch unit {
	case "ms":
		unitDur = time.Millisecond
	case "ns":
		unitDur = time.Nanosecond
	case "m":
		unitDur = time.Minute
	case "s":
		unitDur = time.Second
	default:
		panic(fmt.Errorf("%s not a valid unit. Provide either 's' (seconds) or 'm' (minutes)", unit))
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		panic(fmt.Errorf("failed to convert amount to int: %w", err))
	}

	return time.Duration(amount) * unitDur
}

func (a *App) Start() {
	wg := &sync.WaitGroup{}

	a.handleExit(wg)
	a.watchMemoryUsage(wg)
	a.watchExecutable(wg)
	wg.Wait()
}

func (a *App) watchExecutable(wg *sync.WaitGroup) {
	if !a.runsExecutable {
		return
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.executable.Wait()
		a.cancel()
	}()
}

func (a *App) watchMemoryUsage(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer a.cancel()
		ch := a.process.WatchMemoryUsage(a.refreshInterval)
	LOOP:
		for {
			select {
			case memUsage, ok := <-ch:
				if !ok {
					break LOOP
				}
				fmt.Printf("memory usage: %d mb\n", memUsage.Rss/1024)
				a.memExtractor.Add(extractors.MemoryUsageData{
					Rss:       memUsage.Rss,
					RssSwap:   memUsage.RssSwap,
					Timestamp: time.Now(),
				})
				if memUsage.Rss > a.peakMem {
					a.peakMem = memUsage.Rss
				}
			case <-a.ctx.Done():
				a.writeFiles()
				break LOOP
			}
		}
	}()
}

func (a *App) writeFiles() {
	err := a.memExtractor.StopAndExtract()
	if err != nil {
		panic(fmt.Errorf("failed writing files: %w", err))
	}
}

func (a *App) handleExit(wg *sync.WaitGroup) {
	wg.Add(1)
	startTime := time.Now()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		defer wg.Done()
	LOOP:
		for {
			select {
			case <-c:
				a.cancel()
				break LOOP
			case <-a.ctx.Done():
				break LOOP
			}
		}
		a.printPeakMemory()
		totalTime := time.Since(startTime)
		fmt.Println(totalTime)
	}()
}

func (a *App) printPeakMemory() {
	fmt.Printf("\npeak memory: %d mb\n", a.peakMem/1024)
}
