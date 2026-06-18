//go:build darwin

package p2p

import "os"

func isProcessPrivileged() bool {
	return os.Geteuid() == 0
}

func isPEDLLArchitectureCompatible(string) (bool, string) {
	return true, "not required on this platform"
}
