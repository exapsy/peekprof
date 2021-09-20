package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	flag.Usage = func() {
		usage := fmt.Sprintf(`Usage: %s {-pid <pid>|-cmd <command>} [-out <html output>] [-printoutput] [-refresh <integer>{ns|ms|s|m}]`, os.Args[0])
		fmt.Println(usage)
	}

	pidPtr := flag.Int("pid", 0, "PID of the process")
	cmdPtr := flag.String("cmd", "", "Command to run")
	outPtr := flag.String("out", "out.html", "HTML file output path for the line chart")
	refreshInterval := flag.String("refresh", "1s", "The interval at which it refreshes the stats of the process")
	printOutput := flag.Bool("printoutput", false, "Print the command's stdout and stderr")
	parent := flag.Bool("parent", false, "benchmark the parent of the process and all its children, only when no cmd is specified")
	force := flag.Bool("f", false, "force even if the command has errors. This is useful when attempting to benchmark parent but no parent exists")

	flag.Parse()

	var ecmd *exec.Cmd // The command executed if -pid is not given
	usePid := false    // Inspect another running process if true

	if pidPtr != nil && *pidPtr > 1 {
		usePid = true
	}
	if usePid {
		if *parent {
			ppid, err := getParentPid(*pidPtr)
			if !*force {
				if err != nil {
					panic(fmt.Errorf("failed getting parent pid: %v", err))
				} else if ppid == 0 {
					panic(fmt.Errorf("parent id is 0"))
				}
			}
			if ppid > 0 {
				pidPtr = &ppid
			}
		}
		fmt.Printf("pid: %d\n", *pidPtr)
	} else {
		if cmdPtr == nil || *cmdPtr == "" {
			flag.Usage()
			return
		}

		args := strings.Fields(*cmdPtr)
		ecmd = exec.Command(args[0], args[1:]...)
		if *printOutput {
			ecmd.Stdout = NewCommandStdout()
			ecmd.Stderr = NewCommandStderr()
		}
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

func getParentPid(pid int) (int, error) {
	out, err := exec.Command(
		"bash", "-c",
		fmt.Sprintf(`cat /proc/%d/status | grep PPid | sed "s/^PPid:\ *\t*\([0-9]*\)/\1/"`, pid),
	).Output()
	if err != nil {
		return 0, fmt.Errorf("process has no parent: %w", err)
	}
	ppid, err := strconv.Atoi(strings.Trim(string(out), "\n "))
	if err != nil {
		return 0, fmt.Errorf("failed to convert pid to int: %s", err)
	}
	return ppid, nil
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
