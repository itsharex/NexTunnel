//go:build !linux

package main

import (
	"encoding/json"
	"os"
	"runtime"
	"time"
)

type verifyReport struct {
	GeneratedAt string        `json:"generated_at"`
	Passed      bool          `json:"passed"`
	Checks      []checkResult `json:"checks"`
}

type checkResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func main() {
	report := verifyReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Passed:      false,
		Checks: []checkResult{
			{
				Name:   "linux_required",
				Passed: false,
				Detail: "eBPF XDP verification requires Linux; current platform is " + runtime.GOOS,
			},
		},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	os.Exit(1)
}
