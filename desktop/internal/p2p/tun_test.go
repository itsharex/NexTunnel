package p2p

import (
	"net"
	"testing"
)

func TestTUNConfig_Defaults(t *testing.T) {
	cfg := DefaultTUNConfig()

	if cfg.Name != "nextunnel0" {
		t.Errorf("Name = %q, want %q", cfg.Name, "nextunnel0")
	}
	if cfg.MTU != 1420 {
		t.Errorf("MTU = %d, want 1420", cfg.MTU)
	}
	if !cfg.LocalIP.Equal(net.ParseIP("10.7.0.1")) {
		t.Errorf("LocalIP = %s, want 10.7.0.1", cfg.LocalIP)
	}
	if !cfg.PeerIP.Equal(net.ParseIP("10.7.0.2")) {
		t.Errorf("PeerIP = %s, want 10.7.0.2", cfg.PeerIP)
	}
	if cfg.Subnet == nil {
		t.Error("Subnet is nil")
	}

	t.Logf("TUN defaults: name=%s mtu=%d local=%s peer=%s subnet=%s",
		cfg.Name, cfg.MTU, cfg.LocalIP, cfg.PeerIP, cfg.Subnet)
}

func TestNetTun_TUNDeviceInterface(t *testing.T) {
	tun := newNetTun(0)
	defer tun.Close()

	// Verify interface compliance
	var _ TUNDevice = tun

	// Test MTU
	mtu, err := tun.MTU()
	if err != nil {
		t.Fatalf("MTU: %v", err)
	}
	if mtu != 1420 {
		t.Errorf("MTU = %d, want 1420", mtu)
	}

	// Test Name
	name, err := tun.Name()
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	if name != "netTun" {
		t.Errorf("Name = %q, want %q", name, "netTun")
	}

	// Test IP addresses
	if !tun.LocalAddr().Equal(net.ParseIP("10.7.0.1")) {
		t.Errorf("LocalAddr = %s, want 10.7.0.1", tun.LocalAddr())
	}
	if !tun.PeerAddr().Equal(net.ParseIP("10.7.0.2")) {
		t.Errorf("PeerAddr = %s, want 10.7.0.2", tun.PeerAddr())
	}

	// Test packet roundtrip via WriteFromApp → ReadToApp
	testPkt := []byte("hello tun device")
	if err := tun.WriteFromApp(testPkt); err != nil {
		t.Fatalf("WriteFromApp: %v", err)
	}

	got, err := tun.ReadToApp()
	if err != nil {
		t.Fatalf("ReadToApp: %v", err)
	}
	if string(got) != string(testPkt) {
		t.Errorf("roundtrip: got %q, want %q", got, testPkt)
	}

	t.Logf("TUN device: name=%s mtu=%d local=%s peer=%s roundtrip OK",
		name, mtu, tun.LocalAddr(), tun.PeerAddr())
}

func TestPlatformCapabilities(t *testing.T) {
	caps := CurrentPlatform()

	t.Logf("Platform: %s, KernelTUN=%v, UserspaceNetstack=%v, NeedsAdmin=%v",
		caps.PlatformName, caps.HasKernelTUN, caps.HasUserspaceNetstack, caps.NeedsAdminPrivilege)

	// Userspace netstack should always be available
	if !caps.HasUserspaceNetstack {
		t.Error("HasUserspaceNetstack should be true")
	}
}
