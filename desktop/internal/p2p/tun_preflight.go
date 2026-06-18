package p2p

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	ProductionModeKernelTUN   = "kernel_tun"
	ProductionModeUserspace   = "userspace_tun"
	ProductionModeP2POnly     = "p2p_only"
	ProductionModeUnsupported = "unsupported"

	IssueSeverityBlocker = "blocker"
	IssueSeverityWarning = "warning"
	IssueSeverityInfo    = "info"
)

// PlatformIssue 描述生产数据面的一项前置条件或降级说明。
type PlatformIssue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}

type tunPreflightInput struct {
	platformName      string
	hasKernelSupport  bool
	hasUserspaceStack bool
	needsPrivilege    bool
	privileged        bool
	wintun            wintunPreflightResult
	linuxTunAvailable bool
}

type wintunPreflightResult struct {
	required       bool
	found          bool
	path           string
	archCompatible bool
	detail         string
}

// CurrentPlatform returns the capabilities of the current platform.
func CurrentPlatform() PlatformCapabilities {
	input := tunPreflightInput{
		platformName:      runtime.GOOS,
		hasKernelSupport:  hasKernelTUNSupport(),
		hasUserspaceStack: true,
		needsPrivilege:    platformNeedsPrivilege(runtime.GOOS),
		privileged:        isProcessPrivileged(),
		wintun:            detectWintunPreflight(),
		linuxTunAvailable: linuxTunDeviceAvailable(),
	}
	return evaluatePlatformCapabilities(input)
}

func evaluatePlatformCapabilities(input tunPreflightInput) PlatformCapabilities {
	caps := PlatformCapabilities{
		HasKernelTUN:         input.hasKernelSupport,
		HasUserspaceNetstack: input.hasUserspaceStack,
		NeedsAdminPrivilege:  input.needsPrivilege && !input.privileged,
		PlatformName:         input.platformName,
		UserspaceModeAllowed: input.hasUserspaceStack,
		BlockingIssues:       make([]PlatformIssue, 0, 3),
		DegradedFeatures:     make([]PlatformIssue, 0, 2),
		RecommendedActions:   make([]string, 0, 3),
	}

	if !input.hasKernelSupport {
		caps.BlockingIssues = append(caps.BlockingIssues, PlatformIssue{
			Code:     "kernel_tun_unsupported",
			Severity: IssueSeverityBlocker,
			Message:  fmt.Sprintf("%s 不支持真实内核 TUN。", input.platformName),
			Action:   "只能使用用户态或 P2P-only 模式，不能宣称系统路由 TUN 生产可用。",
		})
	}

	if input.needsPrivilege && !input.privileged {
		caps.BlockingIssues = append(caps.BlockingIssues, PlatformIssue{
			Code:     "privilege_required",
			Severity: IssueSeverityBlocker,
			Message:  "当前进程缺少创建 TUN 或注入路由所需权限。",
			Action:   privilegeAction(input.platformName),
		})
	}

	if input.platformName == "windows" {
		addWindowsWintunIssues(&caps, input.wintun)
	}
	if input.platformName == "linux" && !input.linuxTunAvailable {
		caps.BlockingIssues = append(caps.BlockingIssues, PlatformIssue{
			Code:     "linux_dev_net_tun_missing",
			Severity: IssueSeverityBlocker,
			Message:  "/dev/net/tun 不存在或不可访问。",
			Action:   "加载 tun 模块，并确保容器/系统授予 /dev/net/tun 与 CAP_NET_ADMIN。",
		})
	}

	if len(caps.BlockingIssues) == 0 && input.hasKernelSupport {
		caps.KernelTUNReady = true
		caps.ProductionMode = ProductionModeKernelTUN
	} else if input.hasUserspaceStack {
		caps.ProductionMode = ProductionModeP2POnly
		caps.DegradedFeatures = append(caps.DegradedFeatures, PlatformIssue{
			Code:     "userspace_tun_fallback",
			Severity: IssueSeverityWarning,
			Message:  "用户态 netTun 只能用于测试或受限模式，不会接管系统路由。",
			Action:   "修复阻塞项后再启用真实 TUN；否则只使用 P2P/Relay 转发能力。",
		})
	} else {
		caps.ProductionMode = ProductionModeUnsupported
	}

	for _, issue := range caps.BlockingIssues {
		if issue.Action != "" {
			caps.RecommendedActions = append(caps.RecommendedActions, issue.Action)
		}
	}
	for _, issue := range caps.DegradedFeatures {
		if issue.Action != "" {
			caps.RecommendedActions = append(caps.RecommendedActions, issue.Action)
		}
	}
	return caps
}

func addWindowsWintunIssues(caps *PlatformCapabilities, wintun wintunPreflightResult) {
	if !wintun.required {
		return
	}
	if !wintun.found {
		caps.BlockingIssues = append(caps.BlockingIssues, PlatformIssue{
			Code:     "wintun_dll_missing",
			Severity: IssueSeverityBlocker,
			Message:  "未找到 wintun.dll。",
			Action:   "安装包需随附与进程架构匹配的 wintun.dll，并放在 EXE 同目录或 System32。",
		})
		return
	}
	if !wintun.archCompatible {
		caps.BlockingIssues = append(caps.BlockingIssues, PlatformIssue{
			Code:     "wintun_dll_arch_mismatch",
			Severity: IssueSeverityBlocker,
			Message:  "wintun.dll 架构与当前进程不匹配。",
			Action:   "替换为 amd64/arm64 等与当前进程一致的官方 wintun.dll。",
		})
		return
	}
	caps.DegradedFeatures = append(caps.DegradedFeatures, PlatformIssue{
		Code:     "wintun_dll_ready",
		Severity: IssueSeverityInfo,
		Message:  "wintun.dll 已就绪。",
		Action:   wintun.path,
	})
}

func platformNeedsPrivilege(goos string) bool {
	switch goos {
	case "windows", "darwin", "linux":
		return true
	default:
		return false
	}
}

func privilegeAction(goos string) string {
	switch goos {
	case "windows":
		return "以管理员身份运行 NexTunnel，或由安装器/服务进程创建 Wintun 适配器。"
	case "darwin":
		return "使用授权 helper、LaunchDaemon 或管理员权限创建 utun 并注入路由。"
	case "linux":
		return "使用 root 或授予 CAP_NET_ADMIN，并开放 /dev/net/tun。"
	default:
		return "使用具备系统网络配置权限的运行方式。"
	}
}

func linuxTunDeviceAvailable() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	_, err := os.Stat("/dev/net/tun")
	return err == nil
}

func wintunSearchPaths() []string {
	exePath, _ := os.Executable()
	paths := make([]string, 0, 3)
	if exePath != "" {
		paths = append(paths, filepath.Join(filepath.Dir(exePath), "wintun.dll"))
	}
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		paths = append(paths, filepath.Join(systemRoot, "System32", "wintun.dll"))
	}
	return paths
}

func detectWintunPreflight() wintunPreflightResult {
	if runtime.GOOS != "windows" {
		return wintunPreflightResult{}
	}
	for _, candidate := range wintunSearchPaths() {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		compatible, detail := isPEDLLArchitectureCompatible(candidate)
		return wintunPreflightResult{
			required:       true,
			found:          true,
			path:           candidate,
			archCompatible: compatible,
			detail:         detail,
		}
	}
	return wintunPreflightResult{required: true, detail: "wintun.dll not found in application dir or System32"}
}
