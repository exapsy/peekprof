package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/exapsy/peekprof/internal/extractors"
	httphandler "github.com/exapsy/peekprof/internal/handlers/http"
	"github.com/exapsy/peekprof/internal/process"
)

type App struct {
	process           process.Process
	runsExecutable    bool
	executable        *exec.Cmd
	ctx               context.Context
	cancel            context.CancelFunc
	peakMem           int64
	htmlFilename      string
	csvFilename       string
	refreshInterval   time.Duration
	extractor         extractors.Extractors
	chartLiveUpdates  bool
	host              string
	eventSourceBroker *httphandler.EventSourceServer
	server            *http.Server
}

type AppOptions struct {
	PID              int32
	Host             string
	RunsExecutable   bool
	Cmd              *exec.Cmd
	HtmlFilename     string
	CsvFilename      string
	RefreshInterval  time.Duration
	ChartLiveUpdates bool
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

	if opts.Host == "" {
		opts.Host = "localhost:8089"
	}

	exts := []interface{}{}
	if opts.CsvFilename != "" {
		csvExtractorOpts := extractors.NewCsvExtractorOptions(opts.CsvFilename)
		exts = append(exts, csvExtractorOpts)
	}
	if opts.HtmlFilename != "" {
		chartExtractorOpts := extractors.NewChartExtractorOptions(pname, opts.HtmlFilename)
		if opts.ChartLiveUpdates {
			chartExtractorOpts.UpdateLive(opts.Host)
		}
		exts = append(exts, chartExtractorOpts)
	}

	extractor := extractors.NewExtractors(exts...)

	var esb *httphandler.EventSourceServer
	var server *http.Server
	if opts.ChartLiveUpdates {
		esb = httphandler.NewEventSourceServer()
		h := http.NewServeMux()
		h.Handle("/process/updates", esb)
		server = &http.Server{Addr: opts.Host, Handler: h}
	}

	return &App{
		runsExecutable:    opts.RunsExecutable,
		process:           p,
		ctx:               ctx,
		cancel:            cancel,
		executable:        opts.Cmd,
		peakMem:           0,
		htmlFilename:      opts.HtmlFilename,
		csvFilename:       opts.CsvFilename,
		refreshInterval:   opts.RefreshInterval,
		extractor:         extractor,
		host:              opts.Host,
		chartLiveUpdates:  opts.ChartLiveUpdates,
		eventSourceBroker: esb,
		server:            server,
	}
}

func (a *App) Start() {
	wg := &sync.WaitGroup{}

	a.startHttpServer(wg)
	a.handleExit(wg)
	a.watchMemoryUsage(wg)
	a.watchExecutable(wg)
	wg.Wait()
}

func (a *App) startHttpServer(wg *sync.WaitGroup) {
	if !a.chartLiveUpdates || a.htmlFilename == "" {
		return
	}
	wg.Add(1)
	// add wg.done
	go func() {
		defer wg.Done()
		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		if err != nil {
			panic(err)
		}
	}()
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
		ch := a.process.WatchStats(a.ctx, a.refreshInterval)
	LOOP:
		for {
			select {
			case pstats, ok := <-ch:
				if !ok {
					break LOOP
				}
				fmt.Printf("memory usage: %d mb\tcpu usage: %.1f%%\n", pstats.MemoryUsage.Rss/1024, pstats.CpuUsage.Percentage)
				err := a.extractor.Add(extractors.ProcessStatsData{
					MemoryUsage: extractors.MemoryUsageData{
						Rss:     pstats.MemoryUsage.Rss,
						RssSwap: pstats.MemoryUsage.RssSwap,
					},
					CpuUsage: extractors.CpuUsageData{
						Percentage: pstats.CpuUsage.Percentage,
					},
					Timestamp: time.Now(),
				})
				if err != nil {
					fmt.Printf("error while extracting: %s", err)
				}
				if pstats.MemoryUsage.Rss > a.peakMem {
					a.peakMem = pstats.MemoryUsage.Rss
				}
				if a.chartLiveUpdates {
					pstatsJson, err := json.Marshal(pstats)
					if err != nil {
						panic(fmt.Errorf("[error] could not marshal pstats: %w", err))
					}
					a.eventSourceBroker.Notifier <- pstatsJson
				}
			case <-a.ctx.Done():
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

		if a.chartLiveUpdates {
			// Shut down server
			ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
			defer cancel()
			err := a.server.Shutdown(ctx)
			if errors.Is(err, context.Canceled) {
				// Do nothing
			} else if err != nil {
				panic(fmt.Errorf("failed shutting down server: %w", err))
			}
		}

		a.writeFiles()
		a.printPeakMemory()
		totalTime := time.Since(startTime)
		fmt.Println(totalTime)
	}()
}

func (a *App) printPeakMemory() {
	fmt.Printf("\npeak memory: %d mb\n", a.peakMem/1024)
}
