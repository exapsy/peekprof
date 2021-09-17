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

	"github.com/exapsy/peakben/internal/chart"
	"github.com/exapsy/peakben/internal/process"
)

type App struct {
	process         *process.Process
	runsExecutable  bool
	executable      *exec.Cmd
	ctx             context.Context
	cancel          context.CancelFunc
	peakMem         int64
	outPath         string
	refreshInterval time.Duration
}

type AppOptions struct {
	PID             int32
	RunsExecutable  bool
	Cmd             *exec.Cmd
	Out             string
	RefreshInterval string
}

func NewApp(opts *AppOptions) *App {
	refreshInterval := parseStringToDuration(opts.RefreshInterval)

	p, err := process.NewProcess(opts.PID)
	if err != nil {
		panic(fmt.Sprintf("failed to get process: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		runsExecutable:  opts.RunsExecutable,
		process:         p,
		ctx:             ctx,
		cancel:          cancel,
		executable:      opts.Cmd,
		peakMem:         0,
		outPath:         opts.Out,
		refreshInterval: refreshInterval,
	}
}

// parseStringToDuration parses a string of format <amount><unit> to time.Duration
// Example:
// 2s becomes 2 * time.Second
func parseStringToDuration(s string) time.Duration {
	reg := regexp.MustCompile(`(\d+)(\w)`)
	arr := reg.FindStringSubmatch(s)
	if len(arr) != 3 {
		panic(fmt.Errorf("time %s is not of correct format <amount><unit> (unit: [s: seconds, m: minutes])", s))
	}
	amountStr := arr[1]
	unit := arr[2]
	var amount int
	var unitDur time.Duration

	switch unit {
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
	defer os.Exit(0)
	wg := &sync.WaitGroup{}
	a.catchInterrupt(wg)
	a.watchMemoryUsage(wg)
	a.watchExecutable(wg)
	wg.Wait()
}

func (a *App) watchExecutable(wg *sync.WaitGroup) {
	if !a.runsExecutable {
		return
	}
	go func() {
		a.executable.Wait()
		fmt.Println("Executable has exited")
		wg.Add(-2)
	}()
}

func (a *App) watchMemoryUsage(wg *sync.WaitGroup) {
	wg.Add(1)
	chart := chart.NewMemoryUsageChart()
	go func() {
		defer wg.Done()
		ch := a.process.WatchMemoryUsage(a.refreshInterval)
	LOOP:
		for {
			select {
			case memUsage := <-ch:
				fmt.Printf("memory usage: %d mb\n", memUsage.Rss/1024)
				chart.AddValues(memUsage.Rss, memUsage.Vsz)
				if memUsage.Vsz > a.peakMem {
					a.peakMem = memUsage.Vsz
				}
			case <-a.ctx.Done():
				a.writeChart(chart)
				break LOOP
			}
		}
	}()
}

func (a *App) writeChart(chart *chart.MemoryUsageChart) {
	f, err := os.Create(a.outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	chart.StopAndGenerateChart(f)
	fmt.Printf("chart has been written at %s\n", a.outPath)
}

func (a *App) catchInterrupt(wg *sync.WaitGroup) {
	wg.Add(1)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		defer wg.Done()
		for range c {
			a.printPeakMemory()
			a.cancel()
			break
		}
	}()
}

func (a *App) printPeakMemory() {
	fmt.Printf("\npeak memory: %d mb\n", a.peakMem/1024)
}
