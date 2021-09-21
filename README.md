# Peakben

Peakben is a benchmarking tool used to benchmark the **memory usage** of a process.

![Chart](https://user-images.githubusercontent.com/9019120/133746857-cefd82ff-dae9-474f-88e3-748640251936.png)

## Usage

The benchmark is designed to run until the running process terminates. If you wish to terminate benchmark sooner, just interrupt the benchmark and it will safely terminate the program and write the results.

```nosyntax
Usage: peakben {-pid <pid>|-cmd <command>} [-html <filename>] [-csv <filename>] [-printoutput]
  [-refresh <integer>{ns|ms|s|m}] [-printoutput] [-parent] [-force]

  -pid Track a running process

  -cmd Execute a command and track its memory usage

  -html Extract a chart into an HTML file

  -csv Extract timestamped memory data into a csv

  -refresh The interval at which it checks the memory usage of the process
       [default is 1 second]

  -printoutput Print the corresponding output of the process to stdout & stderr
  
  -parent Track the parent of the provided PID. If no parent exists, an error is returned
      unless -force is provided. If -cmd is provided this is ignored.
      
  -force Ignore errors of parent process not existing
```

### Extract CSV and Chart

```sh
peakben -pid 47123 -html out.html -csv out.csv
```

### Get memory usage by PID

```sh
peakben -pid 47123 -html out.html
```

### Get memory usage from a running command

```sh
peakben -cmd="go test -bench=. -benchtime 300x" -out out.html
```

### Change refresh rate

**Refresh every 3 seconds:**

```sh
peakben -pid 53432 -html out.html -refresh 3s
```

**Refresh every 50 nanoseconds:**

```sh
peakben -pid 53432 -html out.html -refresh 50ns
```

### Profile the parent of a process by child pid

```sh
peakben -pid 53432 -parent
```

## Support

Current support is for **Linux** only.
