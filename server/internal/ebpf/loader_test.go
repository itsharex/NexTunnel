package ebpf

import (
	"testing"
)

func TestLoader_GracefulDegradation(t *testing.T) {
	cfg := DefaultEBPFConfig()
	cfg.Enabled = true

	loader := NewLoader(cfg)

	// Load should succeed (degrade on non-Linux)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// On non-Linux, mode should be userspace
	mode := loader.GetMode()
	t.Logf("Forwarding mode: %s", mode)

	// Unload should succeed
	if err := loader.Unload(); err != nil {
		t.Fatalf("Unload: %v", err)
	}
}

func TestLoader_DisabledByConfig(t *testing.T) {
	cfg := DefaultEBPFConfig()
	cfg.Enabled = false

	loader := NewLoader(cfg)
	if err := loader.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loader.GetMode() != ModeUserspace {
		t.Errorf("mode = %q, want %q", loader.GetMode(), ModeUserspace)
	}
}

func TestLoader_Stats(t *testing.T) {
	loader := NewLoader(DefaultEBPFConfig())
	_ = loader.Load()

	// Record some packets
	loader.RecordForward(1500)
	loader.RecordForward(800)
	loader.RecordForward(2000)
	loader.RecordDrop()

	stats := loader.Stats()
	if stats.PacketsForwarded != 3 {
		t.Errorf("PacketsForwarded = %d, want 3", stats.PacketsForwarded)
	}
	if stats.BytesForwarded != 4300 {
		t.Errorf("BytesForwarded = %d, want 4300", stats.BytesForwarded)
	}
	if stats.PacketsDropped != 1 {
		t.Errorf("PacketsDropped = %d, want 1", stats.PacketsDropped)
	}

	t.Logf("Stats: mode=%s packets=%d bytes=%d dropped=%d",
		stats.Mode, stats.PacketsForwarded, stats.BytesForwarded, stats.PacketsDropped)
}

func TestForwardingMode_Values(t *testing.T) {
	if ModeKernel != "kernel" {
		t.Errorf("ModeKernel = %q, want %q", ModeKernel, "kernel")
	}
	if ModeUserspace != "userspace" {
		t.Errorf("ModeUserspace = %q, want %q", ModeUserspace, "userspace")
	}
}

func TestDefaultEBPFConfig(t *testing.T) {
	cfg := DefaultEBPFConfig()
	if !cfg.Enabled {
		t.Error("default Enabled should be true")
	}
	if cfg.InterfaceName != "eth0" {
		t.Errorf("default InterfaceName = %q, want %q", cfg.InterfaceName, "eth0")
	}
	if cfg.XDPMode != "skb" {
		t.Errorf("default XDPMode = %q, want %q", cfg.XDPMode, "skb")
	}
}
