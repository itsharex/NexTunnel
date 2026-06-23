//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/nextunnel/desktop/internal/p2p"
	"golang.org/x/sys/windows"
)

const (
	wintunRepairTimeout  = 45 * time.Second
	maxWintunArchiveSize = 8 << 20

	peMachineI386  = 0x014c
	peMachineAMD64 = 0x8664
	peMachineARM64 = 0xaa64
)

func currentWintunStatus() WintunStatus {
	caps := p2p.CurrentPlatform()
	status := WintunStatus{
		Installable: true,
		NeedsAdmin:  caps.NeedsAdminPrivilege,
		Message:     "未找到 wintun.dll。",
		Action:      "点击“修复 Wintun”下载官方 DLL，或以管理员身份重启后修复。",
	}
	for _, issue := range caps.DegradedFeatures {
		if issue.Code == "wintun_dll_ready" {
			status.Found = true
			status.Path = issue.Action
			status.ArchCompatible = true
			status.Message = issue.Message
			status.Action = issue.Action
			return status
		}
	}
	for _, issue := range caps.BlockingIssues {
		switch issue.Code {
		case "wintun_dll_missing":
			status.Message = issue.Message
			status.Action = issue.Action
		case "wintun_dll_arch_mismatch":
			status.Found = true
			status.ArchCompatible = false
			status.Message = issue.Message
			status.Action = issue.Action
		}
	}
	return status
}

func repairWintun(input RepairWintunInput) (WintunStatus, error) {
	source, err := normalizeWintunRepairSource(input.Source)
	if err != nil {
		return currentWintunStatus(), err
	}
	if source == wintunRepairSourceBundled {
		return currentWintunStatus(), fmt.Errorf("当前安装包未向应用内暴露内置 Wintun 资源，请使用 download 修复")
	}

	executablePath, err := os.Executable()
	if err != nil {
		return currentWintunStatus(), fmt.Errorf("resolve executable path: %w", err)
	}
	targetPath := filepath.Join(filepath.Dir(executablePath), "wintun.dll")
	dllBytes, err := downloadOfficialWintunDLL()
	if err != nil {
		return currentWintunStatus(), err
	}
	if err := writeWintunDLL(targetPath, dllBytes); err != nil {
		return currentWintunStatus(), fmt.Errorf("write wintun.dll: %w", err)
	}

	status := currentWintunStatus()
	if !status.Found || !status.ArchCompatible {
		return status, fmt.Errorf("wintun.dll installed but still not ready: %s", status.Message)
	}
	status.Message = "Wintun 已修复。"
	status.Action = targetPath
	return status, nil
}

func relaunchAsAdminForWintunRepair() error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(executablePath)
	args, _ := windows.UTF16PtrFromString("--repair-wintun")
	cwd, _ := windows.UTF16PtrFromString(filepath.Dir(executablePath))
	return windows.ShellExecute(0, verb, file, args, cwd, windows.SW_HIDE)
}

func runWintunRepairCommandIfRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--repair-wintun" {
			_, err := repairWintun(RepairWintunInput{Source: wintunRepairSourceDownload})
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	return false
}

func downloadOfficialWintunDLL() ([]byte, error) {
	client := &http.Client{Timeout: wintunRepairTimeout}
	resp, err := client.Get(defaultWintunDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("download Wintun: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download Wintun: unexpected status %s", resp.Status)
	}
	zipBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxWintunArchiveSize+1))
	if err != nil {
		return nil, fmt.Errorf("read Wintun archive: %w", err)
	}
	if len(zipBytes) > maxWintunArchiveSize {
		return nil, fmt.Errorf("Wintun archive exceeds %d bytes", maxWintunArchiveSize)
	}
	if err := assertSHA256Bytes(zipBytes, defaultWintunSHA256); err != nil {
		return nil, err
	}
	return extractWintunDLL(zipBytes)
}

func writeWintunDLL(targetPath string, dllBytes []byte) error {
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, dllBytes, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func assertSHA256Bytes(data []byte, expectedHash string) error {
	actualBytes := sha256.Sum256(data)
	actualHash := hex.EncodeToString(actualBytes[:])
	if actualHash != expectedHash {
		return fmt.Errorf("Wintun SHA256 mismatch: expected=%s actual=%s", expectedHash, actualHash)
	}
	return nil
}

func extractWintunDLL(zipBytes []byte) ([]byte, error) {
	archiveDLLPath, err := wintunArchiveDLLPath()
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open Wintun archive: %w", err)
	}
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != archiveDLLPath {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open wintun.dll from archive: %w", err)
		}
		defer rc.Close()
		dllBytes, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("read wintun.dll from archive: %w", err)
		}
		if err := assertWintunDLLMachine(dllBytes); err != nil {
			return nil, err
		}
		return dllBytes, nil
	}
	return nil, fmt.Errorf("wintun.dll not found in official archive: %s", archiveDLLPath)
}

func assertWintunDLLMachine(dllBytes []byte) error {
	machine, err := readPEMachineForRepair(dllBytes)
	if err != nil {
		return err
	}
	expectedMachine, err := expectedWintunMachine()
	if err != nil {
		return err
	}
	if machine != expectedMachine {
		return fmt.Errorf("unexpected wintun.dll architecture: expected=0x%04x actual=0x%04x", expectedMachine, machine)
	}
	return nil
}

func wintunArchiveDLLPath() (string, error) {
	switch runtime.GOARCH {
	case "386":
		return "wintun/bin/x86/wintun.dll", nil
	case "amd64":
		return "wintun/bin/amd64/wintun.dll", nil
	case "arm64":
		return "wintun/bin/arm64/wintun.dll", nil
	default:
		return "", fmt.Errorf("unsupported Windows architecture for Wintun repair: %s", runtime.GOARCH)
	}
}

func expectedWintunMachine() (uint16, error) {
	switch runtime.GOARCH {
	case "386":
		return peMachineI386, nil
	case "amd64":
		return peMachineAMD64, nil
	case "arm64":
		return peMachineARM64, nil
	default:
		return 0, fmt.Errorf("unsupported Windows architecture for Wintun repair: %s", runtime.GOARCH)
	}
}

func readPEMachineForRepair(data []byte) (uint16, error) {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return 0, fmt.Errorf("invalid PE header")
	}
	peOffset := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	if peOffset < 0 || peOffset+6 > len(data) {
		return 0, fmt.Errorf("invalid PE offset")
	}
	if string(data[peOffset:peOffset+4]) != "PE\x00\x00" {
		return 0, fmt.Errorf("invalid PE signature")
	}
	return binary.LittleEndian.Uint16(data[peOffset+4 : peOffset+6]), nil
}
