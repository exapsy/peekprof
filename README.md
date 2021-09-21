# Peakben

Peakben is a benchmarking tool used to benchmark the **memory usage** of a process.

![Chart](https://user-images.githubusercontent.com/9019120/133746857-cefd82ff-dae9-474f-88e3-748640251936.png)

## Usage

The benchmark is designed to run until the running process terminates. If you wish to terminate benchmark sooner, just interrupt the benchmark and it will safely terminate the program and write the results.

```md
Usage: peakbench {-pid <pid>|-cmd <command>} [-html <filename>] [-csv <filename>] [-printoutput] [-refresh <integer>{ns|ms|s|m}]
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
