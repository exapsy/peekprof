package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	flag.Usage = func() {
		usage := fmt.Sprintf(`Usage: %s {-pid <pid>|-cmd <command>} [-html <filename>] [-csv <filename>] [-printoutput]
		[-refresh <integer>{ns|ms|s|m}] [-prc-output] [-parent] [-live] [-livehost <host>] [nooutput]

Output

		The output depends on if you've set the flag -pretty or not.

		With pretty:

		parent id: 5312                                           # (only if -parent is used)
		command id: 5312                                          # (only if -cmd is used)
		00:13:09        memory usage: 26 mb      cpu usage: 8.2%% # Loop
		peak memory: 2 mb                                         # Print peak memory
		20.852955893s                                             # Print profiling time

		Without pretty (csv friendly except two last lines):

		timestamp, rss kb, %%cpu                                           # Print csv heading
		2021-10-04 00:14:12.635316944 +0300 EEST m=+11.947601412,2956,0.0  # Loop
		peak memory: 2 mb                                                  # Print peak memory
		20.852955893s                                                      # Print profiling time


Flags

		-pid Track a running process

		-cmd Execute a command and track its memory usage

		-html Extract a chart into an HTML file

		-csv Extract timestamped memory data into a csv

		-refresh The interval at which it checks the memory usage of the process
							[default is 100ms]
		
		-live Combined with -html provides an html file that listens live updates for the process' stats.
							[default is true]

		-livehost Is the host at which the local running server is running. This is used with -live and -html.
							[default is localhost:8089]

		-pssoutput Print the corresponding output of the process to stdout & stderr
		
		-parent Track the parent of the provided PID. If no parent exists, an error is returned
						unless -force is provided. If -cmd is provided this is ignored.

		-pretty Print in a more human-friendly - non-csv format, and print the pid of the running process.

		-nooutput Stop printing the profiler's output to console`,

			os.Args[0],
		)
		fmt.Println(usage)
	}

	defaultRefreshInterval := 100 * time.Millisecond

	pidPtr := flag.Int("pid", 0, "Track a process by its PID")
	cmdPtr := flag.String("cmd", "", "Track a command by running it")
	htmlPtr := flag.String("html", "", "Extract a chart into an HTML file")
	csvPtr := flag.String("csv", "", "Extract timestamped memory data into a csv")
	refreshInterval := flag.Duration("refresh", defaultRefreshInterval, "The interval at which it checks the memory usage of the process [default is"+defaultRefreshInterval.String()+"]")
	printPssOutput := flag.Bool("prc-output", false, "Print the command's stdout and stderr")
	parent := flag.Bool("parent", false, "Profile the parent of the process and all its children, only when no cmd is specified")
	noOutput := flag.Bool("nooutput", false, "Stop printing the profiler's output to console")
	live := flag.Bool("live", true, "Combined with -html provides an html file that listens live updates for the process' stats")
	livehost := flag.String("livehost", "localhost:8089", `Is the host at which the local running server is running.
		This is used with -live and -html. The profiler automatically opens the file in your browser.
	`)
	pretty := flag.Bool("pretty", false, "Print in a more human-friendly - non-csv format, and print the pid of the running process.")
	showConsole := flag.Bool("console", true, "Show the console output of the process")

	flag.Parse()

	var ecmd *exec.Cmd // The command executed if -pid is not given
	usePid := false    // Inspect another running process if true

	if *cmdPtr == "" && *pidPtr <= 0 {
		fmt.Println("A PID or a command should be specified")
		flag.Usage()
		return
	}

	if *pidPtr > 1 {
		usePid = true
	}
	if usePid {
		if *parent {
			if runtime.GOOS != "linux" {
				panic("-parent is currently supported only in Linux")
			}
			ppid, err := getParentPid(*pidPtr)
			if err != nil {
				panic(fmt.Errorf("failed getting parent pid: %v", err))
			}
			if ppid <= 0 {
				panic(fmt.Errorf("parent id is non positive"))
			}
			if ppid > 0 {
				pidPtr = &ppid
			}
			if *pretty {
				fmt.Printf("parent pid: %d\n", *pidPtr)
			}
		}
	} else {
		if *cmdPtr == "" {
			flag.Usage()
			return
		}

		args := strings.Fields(*cmdPtr)
		ecmd = exec.Command(args[0], args[1:]...)
		if *printPssOutput {
			ecmd.Stdout = NewCommandStdout()
			ecmd.Stderr = NewCommandStderr()
		}
		err := ecmd.Start()
		if err != nil {
			fmt.Printf("failed to start command: %s\n", err)
			os.Exit(1)
		}

		pidPtr = &ecmd.Process.Pid
		if *pretty {
			fmt.Printf("running command pid: %d\n", *pidPtr)
		}
	}

	a := NewApp(&AppOptions{
		PID:              int32(*pidPtr),
		RunsExecutable:   !usePid,
		Cmd:              ecmd,
		HtmlFilename:     *htmlPtr,
		CsvFilename:      *csvPtr,
		RefreshInterval:  *refreshInterval,
		Host:             *livehost,
		ChartLiveUpdates: *live,
		NoProfilerOutput: *noOutput,
		Pretty:           *pretty,
		ShowConsole:      *showConsole,
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
