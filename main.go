package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/hekmon/transmissionrpc/v3"
)

var (
	logLevel        = flag.String("log-level", "info", "Log Level")
	transmissionURL = flag.String("transmission-url", "http://user:password@localhost:9091", "URL of Transmission Server")
	completeRatio   = flag.Int64("ratio", 2, "Required ratio before a finished torrent will be deleted")
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

	endpoint, err := url.Parse(fmt.Sprintf("%s/transmission/rpc", *transmissionURL))
	if err != nil {
		slog.Error("Error parsing Transmission URL", "url", endpoint, "err", err)
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
	slog.Info("Transmission Versions",
		"serverVersion", serverVersion,
		"serverMinVersion", serverMinVersion,
	)

	torrents, err := tbt.TorrentGetAll(ctx)
	if err != nil {
		slog.Error("Error getting all torrents", "err", err)
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
		slog.Info("No completed torrents, exiting", "IDs", removeIDs)
		os.Exit(0)
	}

	slog.Info("Torrent IDs to remove", "IDs", removeIDs, "count", len(removeIDs))
	err = tbt.TorrentRemove(ctx, transmissionrpc.TorrentRemovePayload{
		IDs:             removeIDs,
		DeleteLocalData: true,
	})
	if err != nil {
		slog.Error("Error removing finished torrents", "err", err)
	}
}
