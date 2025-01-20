# Transmission Prune

Transmission Prune is a little tool that can be run to prune finished Transmission torrents. Can be run on a schedule or on demand, it will remove all finished torrents and the downloaded data, assuming the desired ratio has also been met.

## Usage

Configuration via command line arguments.

```
  -interval duration
        Interval to check for completed torrents if '-wait' is enabled (default 5m0s)
  -log-level string
        Log Level (default "info")
  -ratio int
        Required ratio before a finished torrent will be deleted (default 2)
  -transmission-url string
        URL of Transmission Server, in a format like: 'https://user:password@localhost:9091'
  -wait
        Run continuously and check for completed on a loop
```
