//go:build !windows && !darwin && !linux

package p2p

func isProcessPrivileged() bool {
	return false
}

func isPEDLLArchitectureCompatible(string) (bool, string) {
	return true, "not required on this platform"
}
