//go:build linux

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nextunnel/server/internal/ebpf"
)

const (
	defaultVerifyPort      = 9
	defaultStatsWait       = 2 * time.Second
	defaultMaxKernelRules  = 1024
	defaultStatsInterval   = 500 * time.Millisecond
	defaultForwardedBytes  = 128
	defaultDropSampleCount = 1
)

type checkResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type verifyReport struct {
	GeneratedAt string               `json:"generated_at"`
	Interface   string               `json:"interface"`
	XDPMode     string               `json:"xdp_mode"`
	ObjectPath  string               `json:"object_path"`
	Passed      bool                 `json:"passed"`
	Checks      []checkResult        `json:"checks"`
	Stats       ebpf.ForwardingStats `json:"stats"`
}

func main() {
	report := runVerification()
	writeReportAndExit(report)
}

func runVerification() (report verifyReport) {
	var interfaceName string
	var objectPath string
	var xdpMode string
	var verifyPort int
	var statsWait time.Duration
	var requireKernel bool

	flag.StringVar(&interfaceName, "interface", "eth0", "network interface to attach XDP")
	flag.StringVar(&objectPath, "object", "server/internal/ebpf/xdp_forwarder_bpfel.o", "compiled XDP object path")
	flag.StringVar(&xdpMode, "xdp-mode", "skb", "XDP attach mode: skb, drv, hw, or auto")
	flag.IntVar(&verifyPort, "verify-port", defaultVerifyPort, "TCP destination port used for the temporary DROP rule")
	flag.DurationVar(&statsWait, "stats-wait", defaultStatsWait, "time to keep XDP attached for stats collection")
	flag.BoolVar(&requireKernel, "require-kernel", true, "fail instead of falling back to userspace")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	report = verifyReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Interface:   interfaceName,
		XDPMode:     xdpMode,
		ObjectPath:  objectPath,
		Checks:      make([]checkResult, 0, 6),
	}

	loader := ebpf.NewLoader(ebpf.EBPFConfig{
		Enabled:           true,
		RequireKernelMode: requireKernel,
		InterfaceName:     interfaceName,
		XDPMode:           xdpMode,
		XDPObjectPath:     objectPath,
		MaxKernelRules:    defaultMaxKernelRules,
		StatsInterval:     defaultStatsInterval,
		Logger:            logger,
	})

	rules := ebpf.NewRuleMap()
	if err := loader.ConfigureRuleMap(rules); err != nil {
		report.add("configure_rule_map", false, err.Error())
		return report
	}
	report.add("configure_rule_map", true, "rule sync callback installed")

	if err := loader.Load(); err != nil {
		report.add("load_xdp", false, err.Error())
		return report
	}
	report.add("load_xdp", loader.GetMode() == ebpf.ModeKernel, fmt.Sprintf("mode=%s", loader.GetMode()))

	ctx, cancel := context.WithCancel(context.Background())
	loader.StartStats(ctx)
	defer cancel()
	defer func() {
		if err := loader.Unload(); err != nil {
			report.add("unload_xdp", false, err.Error())
			return
		}
		report.add("unload_xdp", true, "detached")
	}()

	ruleID, err := rules.AddRule(&ebpf.ForwardingRule{
		DstPort:  uint16(verifyPort),
		Protocol: 6,
		Action:   ebpf.ActionDrop,
		Priority: 10,
	})
	if err != nil {
		report.add("sync_drop_rule", false, err.Error())
		return report
	}
	report.add("sync_drop_rule", true, fmt.Sprintf("rule_id=%d dst_port=%d", ruleID, verifyPort))

	loader.RecordForward(defaultForwardedBytes)
	for i := 0; i < defaultDropSampleCount; i++ {
		loader.RecordDrop()
	}
	time.Sleep(statsWait)
	report.Stats = loader.Stats()
	report.add("stats_read", report.Stats.Mode == ebpf.ModeKernel, fmt.Sprintf("mode=%s packets=%d dropped=%d", report.Stats.Mode, report.Stats.PacketsForwarded, report.Stats.PacketsDropped))

	if err := rules.RemoveRule(ruleID); err != nil {
		report.add("remove_drop_rule", false, err.Error())
		return report
	}
	report.add("remove_drop_rule", true, fmt.Sprintf("rule_id=%d", ruleID))

	return report
}

func (r *verifyReport) add(name string, passed bool, detail string) {
	r.Checks = append(r.Checks, checkResult{Name: name, Passed: passed, Detail: detail})
}

func (r *verifyReport) finalize() {
	r.Passed = true
	for _, check := range r.Checks {
		if !check.Passed {
			r.Passed = false
			return
		}
	}
}

func writeReportAndExit(report verifyReport) {
	report.finalize()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	if !report.Passed {
		os.Exit(1)
	}
}
