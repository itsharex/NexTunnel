package virtualnet

import (
	"fmt"
	"strings"
	"testing"
)

type recordingRunner struct {
	commands []string
	failAt   int
}

func (r *recordingRunner) Run(name string, args ...string) error {
	command := name
	if len(args) > 0 {
		command += " " + strings.Join(args, " ")
	}
	r.commands = append(r.commands, command)
	if r.failAt > 0 && len(r.commands) == r.failAt {
		return fmt.Errorf("forced failure")
	}
	return nil
}

func testConfig() Config {
	return Config{
		NodeID:    "node-a",
		VirtualIP: "10.7.0.2",
		Subnet:    "10.7.0.0/24",
		Gateway:   "10.7.0.1",
		Interface: "nextunnel0",
		MTU:       1420,
		Routes: []Route{
			{
				Destination: "10.7.0.0/24",
				Gateway:     "10.7.0.1",
				Interface:   "nextunnel0",
				Metric:      100,
			},
		},
	}
}

func TestManager_ApplyAndReset(t *testing.T) {
	runner := &recordingRunner{}
	manager := NewManager(runner, nil)

	state, err := manager.Apply(testConfig())
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !state.Applied {
		t.Fatal("expected virtual network to be applied")
	}
	if state.VirtualIP != "10.7.0.2" || state.Interface != "nextunnel0" {
		t.Fatalf("unexpected state: %+v", state)
	}
	if len(runner.commands) == 0 {
		t.Fatal("expected commands to be executed")
	}

	state, err = manager.Reset()
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if state.Applied {
		t.Fatal("expected virtual network to be reset")
	}
}

func TestManager_ApplyValidation(t *testing.T) {
	manager := NewManager(&recordingRunner{}, nil)
	cfg := testConfig()
	cfg.VirtualIP = ""

	state, err := manager.Apply(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if state.LastError == "" {
		t.Fatal("expected last error to be recorded")
	}
}

func TestManager_ApplyRejectsInvalidNetworkConfig(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "invalid virtual ip",
			mutate: func(cfg *Config) {
				cfg.VirtualIP = "not-an-ip"
			},
		},
		{
			name: "invalid subnet",
			mutate: func(cfg *Config) {
				cfg.Subnet = "10.7.0.0"
			},
		},
		{
			name: "low mtu",
			mutate: func(cfg *Config) {
				cfg.MTU = 128
			},
		},
		{
			name: "invalid route destination",
			mutate: func(cfg *Config) {
				cfg.Routes[0].Destination = "10.7.0.0"
			},
		},
		{
			name: "missing route gateway",
			mutate: func(cfg *Config) {
				cfg.Routes[0].Gateway = ""
			},
		},
		{
			name: "negative metric",
			mutate: func(cfg *Config) {
				cfg.Routes[0].Metric = -1
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &recordingRunner{}
			manager := NewManager(runner, nil)
			cfg := testConfig()
			tc.mutate(&cfg)

			state, err := manager.Apply(cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if len(runner.commands) != 0 {
				t.Fatalf("validation failure must not run commands, got %v", runner.commands)
			}
			if state.LastError == "" {
				t.Fatal("expected validation error to be recorded")
			}
		})
	}
}

func TestManager_ApplyRecordsCommandFailure(t *testing.T) {
	runner := &recordingRunner{failAt: 1}
	manager := NewManager(runner, nil)

	state, err := manager.Apply(testConfig())
	if err == nil {
		t.Fatal("expected command failure")
	}
	if state.Applied {
		t.Fatal("failed apply must not mark state as applied")
	}
	if state.LastError == "" || len(state.LastCommands) == 0 {
		t.Fatalf("expected failure diagnostics, got %+v", state)
	}
}

func TestBuildWindowsApplyCommands(t *testing.T) {
	commands, err := buildApplyCommands("windows", testConfig())
	if err != nil {
		t.Fatalf("buildApplyCommands: %v", err)
	}
	got := make([]string, 0, len(commands))
	for _, command := range commands {
		got = append(got, command.String())
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "netsh interface ip set address name=nextunnel0 static 10.7.0.2 255.255.255.0") {
		t.Fatalf("missing address command:\n%s", joined)
	}
	if !strings.Contains(joined, "netsh interface ipv4 add route prefix=10.7.0.0/24 interface=nextunnel0 nexthop=10.7.0.1 metric=100 store=active") {
		t.Fatalf("missing route command:\n%s", joined)
	}
}

func TestBuildUnsupportedPlatform(t *testing.T) {
	if _, err := buildApplyCommands("plan9", testConfig()); err == nil {
		t.Fatal("expected unsupported platform error")
	}
}
