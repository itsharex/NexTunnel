package main

import (
	"errors"
	"testing"

	"github.com/nextunnel/desktop/internal/p2p"
)

func TestCoordinationStateStoresValues(t *testing.T) {
	state := newCoordinationState()
	exchange := p2p.CandidateExchange{SessionID: "session-1", Role: "initiator"}
	result := p2p.DirectVerifyResult{Role: "initiator", SessionID: "session-1"}

	state.setCandidates("windows", exchange)
	state.setDirect("windows", result)
	state.setRelay("windows", "done")

	gotExchange, ok := state.getCandidates("windows")
	if !ok || gotExchange.SessionID != "session-1" {
		t.Fatalf("candidate exchange not stored correctly: %+v ok=%t", gotExchange, ok)
	}

	gotResult, ok := state.getDirect("windows")
	if !ok || gotResult.SessionID != "session-1" {
		t.Fatalf("direct result not stored correctly: %+v ok=%t", gotResult, ok)
	}

	if gotRelay, ok := state.getRelay("windows"); !ok || gotRelay != "done" {
		t.Fatalf("relay state not stored correctly: %q ok=%t", gotRelay, ok)
	}
}

func TestRunLocalTUNReportsKernelCreationFailure(t *testing.T) {
	originalCreateKernelTUNDevice := createKernelTUNDevice
	createKernelTUNDevice = func(p2p.TUNConfig) (p2p.TUNDevice, error) {
		return nil, errors.New("forced kernel tun failure")
	}
	t.Cleanup(func() {
		createKernelTUNDevice = originalCreateKernelTUNDevice
	})

	rep := newReport("orchestrator")
	cfg := endpointConfig{
		NodeID:    "windows",
		VirtualIP: "10.77.0.1",
		PeerIP:    "10.77.0.2",
		Subnet:    "10.77.0.0/30",
		Gateway:   "10.77.0.2",
		Route:     "10.77.0.2/32",
		Interface: "NexTunnelVerify",
		MTU:       defaultMTU,
	}

	runLocalTUN(t.Context(), rep, cfg, false)

	snapshot := rep.snapshot(false)
	if len(snapshot.Checks) == 0 {
		t.Fatal("expected TUN check result")
	}
	tunCheck, ok := findCheck(snapshot.Checks, "tun_create")
	if !ok {
		t.Fatalf("expected tun_create check, got %+v", snapshot.Checks)
	}
	if tunCheck.Passed {
		t.Fatalf("kernel TUN should not be available in unit test environment: %+v", tunCheck)
	}
	if tunCheck.Detail == "name=netTun" {
		t.Fatalf("verification must report kernel TUN error instead of userspace fallback: %+v", tunCheck)
	}
}

func TestReportSnapshotFinalizes(t *testing.T) {
	rep := newReport("orchestrator")
	rep.add("check-1", true, "ok")
	rep.add("check-2", false, "failed")

	snapshot := rep.snapshot(true)
	if snapshot.Passed {
		t.Fatal("snapshot should fail when any check fails")
	}
	if len(snapshot.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(snapshot.Checks))
	}
}

func findCheck(checks []checkResult, name string) (checkResult, bool) {
	for _, check := range checks {
		if check.Name == name {
			return check, true
		}
	}
	return checkResult{}, false
}
