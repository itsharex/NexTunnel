package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nextunnel/server/internal/relay"
)

func main() {
	fs := flag.NewFlagSet("relay", flag.ExitOnError)
	cfg := relay.ParseFlags(fs)
	var statsInterval time.Duration
	fs.DurationVar(&statsInterval, "stats-interval", 60*time.Second, "interval for periodic stats logging (0 to disable)")
	fs.Parse(os.Args[1:])

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	server := relay.NewServer(cfg, logger)

	if err := server.Run(); err != nil {
		logger.Error("failed to start relay server", "error", err)
		os.Exit(1)
	}

	// Periodic stats logging
	var statsDone chan struct{}
	if statsInterval > 0 {
		statsDone = make(chan struct{})
		go func() {
			defer close(statsDone)
			ticker := time.NewTicker(statsInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					stats := server.GetStats()
					logger.Info("server stats",
						"clients", stats.Clients,
						"proxies", stats.Proxies,
						"sessions", stats.Sessions,
						"bytesIn", stats.BytesIn,
						"bytesOut", stats.BytesOut)
				case <-server.Done():
					return
				}
			}
		}()
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	// Final stats
	stats := server.GetStats()
	logger.Info("final stats",
		"clients", stats.Clients,
		"proxies", stats.Proxies,
		"sessions", stats.Sessions,
		"bytesIn", stats.BytesIn,
		"bytesOut", stats.BytesOut)
}
