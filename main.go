package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	flag.Usage = func() {
		usage := fmt.Sprintf("Usage: %s <pid | executable path>", os.Args[0])
		fmt.Println(usage)
	}

	pidPtr := flag.Int("pid", 0, "PID of the process")
	cmdPtr := flag.String("cmd", "", "Command to run")
	outPtr := flag.String("out", "out.html", "HTML file output path for the line chart")

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

		ecmd = exec.Command("bash", "-c", *cmdPtr)
		err := ecmd.Start()
		if err != nil {
			panic(fmt.Errorf("failed to start command: %w", err))
		}

		parentProcessPid := &ecmd.Process.Pid

		// Wait to make sure the child is running
		for range time.Tick(time.Millisecond * 1000) {
			break
		}
		// Get the child of "bash -c"
		// we don't want to benchmark "bash -c", but the command it executed
		pidPtr = getChildProcessPid(*parentProcessPid)
	}

	a := NewApp(&AppOptions{
		PID:            int32(*pidPtr),
		RunsExecutable: !usePid,
		Cmd:            ecmd,
		Out:            *outPtr,
	})
	a.Start()
}

func getChildProcessPid(pid int) *int {
	cmd := fmt.Sprintf("pgrep -P %d", pid)
	childrenPids, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		panic(err)
	}
	pidStr := strings.Split(string(childrenPids), "\n")[0]
	pidResult, err := strconv.Atoi(pidStr)
	if err != nil {
		panic(err)
	}

	return &pidResult
}
