package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/hekmon/transmissionrpc/v3"
)

var (
	logLevel        = flag.String("log-level", "info", "Log Level")
	transmissionURL = flag.String("transmission-url", "", "URL of Transmission Server, in a format like: 'https://user:password@localhost:9091'")
	completeRatio   = flag.Int64("ratio", 2, "Required ratio before a finished torrent will be deleted")
	wait            = flag.Bool("wait", false, "Run continuously and check for completed on a loop")
	interval        = flag.Duration("interval", time.Minute*5, "Interval to check for completed torrents if '-wait' is enabled")
)

func main() {
	flag.Parse()
	var ll slog.Level
	switch *logLevel {
	case "debug":
		ll = slog.LevelDebug
	case "info":
		ll = slog.LevelInfo
	case "warning":
		ll = slog.LevelWarn
	case "error":
		ll = slog.LevelError
	default:
		ll = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     ll,
		})))
	slog.Info("Starting Transmission Prune",
		"log-level", *logLevel,
		"transmission-url", *transmissionURL,
		"ratio", *completeRatio,
		"wait", *wait,
		"interval", *interval,
	)

	endpoint, err := url.Parse(fmt.Sprintf("%s/transmission/rpc", *transmissionURL))
	if err != nil || *transmissionURL == "" {
		slog.Error("Error parsing Transmission URL",
			"err", err,
			"transmission-url", *transmissionURL,
			"parsedUrl", endpoint,
		)
		os.Exit(1)
	}
	tbt, err := transmissionrpc.New(endpoint, nil)
	if err != nil {
		slog.Error("Error initialising Transmission client", "err", err, "endpoint", endpoint)
		os.Exit(1)
	}
	ctx := context.Background()
	ok, serverVersion, serverMinVersion, err := tbt.RPCVersion(ctx)
	if !ok {
		slog.Error("Version check not ok",
			"err", err,
			"serverVersion", serverVersion,
			"serverMinVersion", serverMinVersion,
		)
		os.Exit(1)
	}
	slog.Debug("Transmission Versions",
		"serverVersion", serverVersion,
		"serverMinVersion", serverMinVersion,
	)

	deleteCompleted(ctx, tbt)
	if !*wait {
		os.Exit(0)
	}

	t := time.Tick(*interval)
	for _ = range t {
		deleteCompleted(ctx, tbt)
	}
}

func deleteCompleted(ctx context.Context, t *transmissionrpc.Client) {
	torrents, err := t.TorrentGetAll(ctx)
	if err != nil {
		slog.Error("Error getting all torrents", "err", err)
		return
	}

	removeIDs := []int64{}

	for _, torrent := range torrents {
		if torrent.IsFinished == nil && !*torrent.IsFinished {
			continue
		}

		if *torrent.DownloadedEver == 0 {
			continue
		}

		ratio := *torrent.UploadedEver / *torrent.DownloadedEver

		if ratio < *completeRatio {
			continue
		}

		slog.Info("Adding torrent to removal list",
			"name", *torrent.Name,
			"ID", *torrent.ID,
			"DownloadedEver", *torrent.DownloadedEver,
			"UploadedEver", *torrent.UploadedEver,
			"PercentDone", *torrent.PercentDone,
			"ratio", ratio,
		)

		removeIDs = append(removeIDs, *torrent.ID)
	}

	if len(removeIDs) <= 0 {
		slog.Info("No completed torrents", "IDs", removeIDs)
		return
	}

	slog.Info("Torrent IDs to remove", "IDs", removeIDs, "count", len(removeIDs))
	err = t.TorrentRemove(ctx, transmissionrpc.TorrentRemovePayload{
		IDs:             removeIDs,
		DeleteLocalData: true,
	})
	if err != nil {
		slog.Error("Error removing finished torrents", "err", err, "removeIDs", removeIDs)
	}
}
