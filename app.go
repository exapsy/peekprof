package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/exapsy/peekprof/internal/extractors"
	"github.com/exapsy/peekprof/internal/process"
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
	extractor       extractors.Extractors
}

type AppOptions struct {
	PID             int32
	RunsExecutable  bool
	Cmd             *exec.Cmd
	HtmlFilename    string
	CsvFilename     string
	RefreshInterval time.Duration
}

func NewApp(opts *AppOptions) *App {
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
		csvExtractorOpts := extractors.NewCsvExtractorOptions(opts.CsvFilename)
		memExtractors = append(memExtractors, csvExtractorOpts)
	}
	if opts.HtmlFilename != "" {
		chartExtractorOpts := extractors.NewChartExtractorOptions(pname, opts.HtmlFilename)
		memExtractors = append(memExtractors, chartExtractorOpts)
	}

	extractor := extractors.NewExtractors(memExtractors...)

	return &App{
		runsExecutable:  opts.RunsExecutable,
		process:         p,
		ctx:             ctx,
		cancel:          cancel,
		executable:      opts.Cmd,
		peakMem:         0,
		htmlFilename:    opts.HtmlFilename,
		csvFilename:     opts.CsvFilename,
		refreshInterval: opts.RefreshInterval,
		extractor:       extractor,
	}
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
		ch := a.process.WatchStats(a.refreshInterval)
	LOOP:
		for {
			select {
			case pstats, ok := <-ch:
				if !ok {
					break LOOP
				}
				fmt.Printf("memory usage: %d mb\n", pstats.MemoryUsage.Rss/1024)
				a.extractor.Add(extractors.ProcessStatsData{
					MemoryUsage: extractors.MemoryUsageData{
						Rss:     pstats.MemoryUsage.Rss,
						RssSwap: pstats.MemoryUsage.RssSwap,
					},
					CpuUsage: extractors.CpuUsageData{
						Percentage: pstats.CpuUsage.Percentage,
					},
					Timestamp: time.Now(),
				})
				if pstats.MemoryUsage.Rss > a.peakMem {
					a.peakMem = pstats.MemoryUsage.Rss
				}
			case <-a.ctx.Done():
				a.writeFiles()
				break LOOP
			}
		}
	}()
}

func (a *App) writeFiles() {
	err := a.extractor.StopAndExtract()
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
