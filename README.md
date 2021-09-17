# Peakben

Peakben is a benchmarking tool used to benchmark the **memory usage** of a process.

![image](https://user-images.githubusercontent.com/9019120/133746857-cefd82ff-dae9-474f-88e3-748640251936.png)

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
