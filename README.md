# Peekprof

Peekprof is a profiling tool used to profile process.

![Chart page](https://user-images.githubusercontent.com/9019120/134160444-e0db5160-14a5-460f-8d39-2737e246482d.png)

## Usage

The profiling is designed to run until the running process terminates. If you wish to terminate the profiler sooner, just interrupt the process and it will safely terminate the program and write the results.

```nosyntax
Usage: peekprof {-pid <pid>|-cmd <command>} [-html <filename>] [-csv <filename>] [-printoutput]
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
