package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/exapsy/peakben/internal/chart"
	"github.com/exapsy/peakben/internal/process"
)

type App struct {
	process        *process.Process
	runsExecutable bool
	executable     *exec.Cmd
	ctx            context.Context
	cancel         context.CancelFunc
	peakMem        int64
	outPath        string
}

type AppOptions struct {
	PID            int32
	RunsExecutable bool
	Cmd            *exec.Cmd
	Out            string
}

func NewApp(opts *AppOptions) *App {
	p, err := process.NewProcess(opts.PID)
	if err != nil {
		panic(fmt.Sprintf("failed to get process: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		runsExecutable: opts.RunsExecutable,
		process:        p,
		ctx:            ctx,
		cancel:         cancel,
		executable:     opts.Cmd,
		peakMem:        0,
		outPath:        opts.Out,
	}
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
		ch := a.process.WatchMemoryUsage(1 * time.Second)
	LOOP:
		for {
			select {
			case memUsage := <-ch:
				fmt.Printf("memory usage: %d mb\n", memUsage/1024)
				chart.AddValue(memUsage)
				if memUsage > a.peakMem {
					a.peakMem = memUsage
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
