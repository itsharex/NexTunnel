//go:build windows

package p2p

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	peMachineI386  = 0x014c
	peMachineAMD64 = 0x8664
	peMachineARM64 = 0xaa64
)

func isProcessPrivileged() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	var elevation uint32
	var returned uint32
	err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &returned)
	return err == nil && elevation != 0
}

func isPEDLLArchitectureCompatible(path string) (bool, string) {
	machine, err := readPEMachine(path)
	if err != nil {
		return false, err.Error()
	}
	expected, err := expectedPEMachine()
	if err != nil {
		return false, err.Error()
	}
	if machine != expected {
		return false, fmt.Sprintf("expected=0x%04x actual=0x%04x", expected, machine)
	}
	return true, fmt.Sprintf("machine=0x%04x", machine)
}

func expectedPEMachine() (uint16, error) {
	switch runtime.GOARCH {
	case "386":
		return peMachineI386, nil
	case "amd64":
		return peMachineAMD64, nil
	case "arm64":
		return peMachineARM64, nil
	default:
		return 0, fmt.Errorf("unsupported windows arch %s", runtime.GOARCH)
	}
}

func readPEMachine(path string) (uint16, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read PE file: %w", err)
	}
	return readPEMachineBytes(data)
}

func readPEMachineBytes(data []byte) (uint16, error) {
	if len(data) < 0x40 {
		return 0, fmt.Errorf("PE file is too small")
	}
	if data[0] != 'M' || data[1] != 'Z' {
		return 0, fmt.Errorf("invalid DOS header")
	}
	peOffset := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	if peOffset < 0 || peOffset+6 > len(data) {
		return 0, fmt.Errorf("invalid PE header offset")
	}
	if string(data[peOffset:peOffset+4]) != "PE\x00\x00" {
		return 0, fmt.Errorf("invalid PE signature")
	}
	return binary.LittleEndian.Uint16(data[peOffset+4 : peOffset+6]), nil
}
