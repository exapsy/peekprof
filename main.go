package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/trace"
	"strings"
)

func main() {
	traceFile, _ := os.Open("./trace.out")
	defer traceFile.Close()
	trace.Start(traceFile)
	defer trace.Stop()

	flag.Usage = func() {
		usage := fmt.Sprintf("Usage: %s <pid | executable path>", os.Args[0])
		fmt.Println(usage)
	}

	pidPtr := flag.Int("pid", 0, "PID of the process")
	cmdPtr := flag.String("cmd", "", "Command to run")
	outPtr := flag.String("out", "out.html", "HTML file output path for the line chart")
	refreshInterval := flag.String("refresh", "1s", "The interval at which it refreshes the stats of the process")

	flag.Parse()

	var ecmd *exec.Cmd // The command executed if -pid is not given
	usePid := false    // Inspect another running process if true

	if pidPtr != nil && *pidPtr > 1 {
		usePid = true
	}
	if usePid {
		fmt.Printf("pid: %d\n", *pidPtr)
	} else {
		if cmdPtr == nil || *cmdPtr == "" {
			flag.Usage()
			return
		}

		args := strings.Fields(*cmdPtr)
		ecmd = exec.Command(args[0], args[1:]...)
		// ecmd.Stdout = NewCommandStdout()
		// ecmd.Stderr = NewCommandStderr()
		err := ecmd.Start()
		if err != nil {
			fmt.Printf("failed to start command: %s\n", err)
			os.Exit(1)
		}

		pidPtr = &ecmd.Process.Pid
		fmt.Printf("running command pid: %d\n", *pidPtr)
	}

	a := NewApp(&AppOptions{
		PID:             int32(*pidPtr),
		RunsExecutable:  !usePid,
		Cmd:             ecmd,
		Out:             *outPtr,
		RefreshInterval: *refreshInterval,
	})
	a.Start()
}

type CommandStdout struct{}

func NewCommandStdout() *CommandStdout {
	return &CommandStdout{}
}

func (c *CommandStdout) Write(b []byte) (int, error) {
	resetColor := []byte("\033[0m\n")
	tag := []byte("\n\033[1;34m[stdout]\033[0m\t")
	accentColor := []byte("\033[34m")
	out := append(tag, accentColor...)
	out = append(out, b...)
	out = append(out, resetColor...)
	n, err := os.Stdout.Write(out)
	return n, err
}

type CommandErr struct{}

func NewCommandStderr() *CommandErr {
	return &CommandErr{}
}

func (c *CommandErr) Write(b []byte) (int, error) {
	tag := []byte("\n\033[1;31m[err]\t")
	accentColor := []byte("\033[31m\033[0m")
	resetColor := []byte("\033[0m\n")
	out := append(tag, accentColor...)
	out = append(out, b...)
	out = append(out, resetColor...)
	n, err := os.Stderr.Write(out)
	return n, err
}
