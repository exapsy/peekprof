![Available in AUR](https://img.shields.io/aur/version/peekprof?style=flat-square)

# Peekprof

Get the CPU and Memory usage of a **single process**, monitor it live, and extract it in CSV and HTML. Get the best out of your optimizations.

<p align="center">
  <img width="500" height="612" src="https://user-images.githubusercontent.com/9019120/134412870-d1713c72-a64f-419a-85c2-1fc67f5471a8.gif">
</p>

## Usage

The profiling is designed to run until the running process terminates. If you wish to terminate the profiler sooner, just interrupt the process and it will safely terminate the program and write the results.

```nosyntax
Usage: peekprof {-pid <pid>|-cmd <command>} [-html <filename>] [-csv <filename>] [-printoutput]
  [-refresh <integer>{ns|ms|s|m}] [-pssoutput] [-parent] [-live] [-livehost <host>] [-nooutput]

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

  -nooutput Stop printing the profiler's output to console`,
```

### Extract CSV and Chart

```sh
peekprof -pid 47123 -html out.html -csv out.csv
```

### Get memory usage by PID

```sh
peekprof -pid 47123
```

### Get memory usage from a running command

```sh
peekprof -cmd="go test -bench=. -benchtime 300x"
```

### Change refresh rate

```sh
peekprof -pid 53432 -refresh 50ms # Refresh every 50 milliseconds
```

### Profile the parent of a process by child pid

```sh
peekprof -pid 53432 -parent
```

## Support

Current support is for **Linux** and **OSX**.

### OSX differences

- Swap is not currenty supported, thus it is not shown either in the extracted files.
- `-parent` is supported only in Linux
- In Linux, the process and process' children metrics are tracked. Currently this behavior is not implemented for OSX.

### License

[MIT](https://github.com/exapsy/peekprof/blob/master/LICENSE) © Apostolis Anastasiou
