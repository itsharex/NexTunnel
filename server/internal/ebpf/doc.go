// Package ebpf provides optional kernel-level packet forwarding acceleration
// for NexTunnel relay servers using eBPF/XDP on Linux.
//
// On non-Linux platforms, the module gracefully degrades to userspace forwarding.
package ebpf
