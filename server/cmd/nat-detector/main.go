package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nextunnel/server/internal/natdetect"
)

func main() {
	fs := flag.NewFlagSet("nat-detector", flag.ExitOnError)
	cfg := natdetect.DefaultConfig()
	fs.StringVar(&cfg.PrimaryAddr, "primary-addr", cfg.PrimaryAddr, "primary IP address to bind")
	fs.StringVar(&cfg.AltAddr, "alt-addr", cfg.AltAddr, "alternate IP address to bind")
	fs.IntVar(&cfg.Port, "port", cfg.Port, "UDP port to listen on")
	fs.StringVar(&cfg.Realm, "realm", cfg.Realm, "STUN/TURN realm")
	fs.Parse(os.Args[1:])

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	server := natdetect.NewServer(cfg, logger)

	if err := server.Start(); err != nil {
		logger.Error("failed to start NAT detection server", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx

	server.Stop()
}
