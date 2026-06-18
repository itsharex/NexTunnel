//go:build linux

package p2p

import (
	"os"
	"strconv"
	"strings"
)

const linuxCapNetAdmin = 12

func isProcessPrivileged() bool {
	if os.Geteuid() == 0 {
		return true
	}
	return hasLinuxCapability(linuxCapNetAdmin)
}

func isPEDLLArchitectureCompatible(string) (bool, string) {
	return true, "not required on this platform"
}

func hasLinuxCapability(capability int) bool {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	return statusHasLinuxCapability(string(data), capability)
}

func statusHasLinuxCapability(status string, capability int) bool {
	for _, line := range strings.Split(status, "\n") {
		if !strings.HasPrefix(line, "CapEff:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return false
		}
		value, err := strconv.ParseUint(fields[1], 16, 64)
		if err != nil {
			return false
		}
		return value&(uint64(1)<<capability) != 0
	}
	return false
}
