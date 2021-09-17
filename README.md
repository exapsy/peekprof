# Peakben

Peakben is a benchmarking tool used to benchmark the **memory usage** of a process.

## Usage

### Get memory usage by PID

```sh
peakben -pid 47123 -out out.html
```

### Get memory usage from a running command

```sh
peakben -cmd="go test -bench=. -benchtime 300x" -out out.html
```

## Support

Current support is for **Linux** only.
