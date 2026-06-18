//go:build linux

package p2p

import "testing"

func TestStatusHasLinuxCapability(t *testing.T) {
	status := "Name:\tnextunnel\nCapEff:\t0000000000001000\n"
	if !statusHasLinuxCapability(status, linuxCapNetAdmin) {
		t.Fatal("expected CAP_NET_ADMIN bit to be detected")
	}

	status = "Name:\tnextunnel\nCapEff:\t0000000000000000\n"
	if statusHasLinuxCapability(status, linuxCapNetAdmin) {
		t.Fatal("did not expect CAP_NET_ADMIN bit")
	}
}
