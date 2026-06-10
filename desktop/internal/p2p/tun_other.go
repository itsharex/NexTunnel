//go:build !linux && !darwin && !windows

package p2p

import (
	"fmt"
)

// createKernelTUN returns an error on unsupported platforms.
func createKernelTUN(cfg TUNConfig) (TUNDevice, error) {
	return nil, fmt.Errorf("kernel TUN not supported on this platform; use userspace netTun instead")
}
