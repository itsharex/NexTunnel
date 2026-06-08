package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nextunnel/server/internal/controlplane"
)

func main() {
	fs := flag.NewFlagSet("control-plane", flag.ExitOnError)
	cfg := controlplane.DefaultControlPlaneConfig()
	fs.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "HTTP API listen address")
	fs.StringVar(&cfg.APIToken, "api-token", cfg.APIToken, "optional Bearer token for control plane HTTP APIs")
	fs.StringVar(&cfg.StorePath, "store-path", cfg.StorePath, "SQLite database path for persistent storage (empty = in-memory)")
	fs.Parse(os.Args[1:])

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	store, err := controlplane.NewStoreFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create store", "error", err)
		os.Exit(1)
	}
	// Close SQLite store on exit if applicable
	if closer, ok := store.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	srv := controlplane.NewServer(cfg, store, controlplane.WithCPLogger(logger))

	if err := srv.Start(); err != nil {
		logger.Error("failed to start control plane", "error", err)
		os.Exit(1)
	}

	logger.Info("NexTunnel Control Plane started", "addr", cfg.ListenAddr)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	_ = context.Background() // reserved for graceful shutdown context
	srv.Stop()

	logger.Info("NexTunnel Control Plane stopped")
}
