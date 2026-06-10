package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nextunnel/pkg/tlsutil"
	"github.com/nextunnel/server/internal/controlplane"
)

func main() {
	fs := flag.NewFlagSet("control-plane", flag.ExitOnError)
	cfg := controlplane.DefaultControlPlaneConfig()
	fs.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "HTTP API listen address")
	fs.StringVar(&cfg.APIToken, "api-token", cfg.APIToken, "optional Bearer token for control plane HTTP APIs")
	fs.StringVar(&cfg.StorePath, "store-path", cfg.StorePath, "SQLite database path for persistent storage (empty = in-memory)")
	fs.StringVar(&cfg.AuditLogPath, "audit-log", cfg.AuditLogPath, "JSON Lines audit log path (empty = disabled)")
	fs.BoolVar(&cfg.IPAMEnabled, "ipam-enabled", cfg.IPAMEnabled, "enable virtual IP allocation for registered nodes")
	fs.StringVar(&cfg.VirtualSubnet, "virtual-subnet", cfg.VirtualSubnet, "virtual network CIDR for node IPAM")
	fs.StringVar(&cfg.VirtualGateway, "virtual-gateway", cfg.VirtualGateway, "virtual network gateway IP")
	fs.StringVar(&cfg.VirtualInterface, "virtual-interface", cfg.VirtualInterface, "virtual TUN interface name advertised to clients")
	fs.IntVar(&cfg.VirtualMTU, "virtual-mtu", cfg.VirtualMTU, "virtual TUN interface MTU")
	fs.IntVar(&cfg.VirtualRouteMetric, "virtual-route-metric", cfg.VirtualRouteMetric, "route metric advertised to clients")

	var tlsCA, tlsCert, tlsKey string
	fs.StringVar(&tlsCA, "tls-ca", "", "CA certificate PEM for mTLS (enables TLS when set)")
	fs.StringVar(&tlsCert, "tls-cert", "", "server certificate PEM for mTLS")
	fs.StringVar(&tlsKey, "tls-key", "", "server private key PEM for mTLS")
	fs.Parse(os.Args[1:])

	if tlsCA != "" && tlsCert != "" && tlsKey != "" {
		cfg.TLSEnabled = true
		cfg.TLS = tlsutil.TLSConfig{CACert: tlsCA, Cert: tlsCert, Key: tlsKey}
	}

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
