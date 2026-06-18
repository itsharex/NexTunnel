package p2p

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// CandidateExchange 是跨进程/跨主机交换 ICE 候选的最小 JSON 结构。
type CandidateExchange struct {
	SessionID  string      `json:"session_id"`
	Role       string      `json:"role"`
	Candidates []Candidate `json:"candidates"`
}

// DirectVerifyResult 描述一次 ICE + UDP punch 直连验证结果。
type DirectVerifyResult struct {
	Role           string `json:"role"`
	SessionID      string `json:"session_id"`
	LocalAddr      string `json:"local_addr"`
	RemoteAddr     string `json:"remote_addr"`
	SelectedLocal  string `json:"selected_local"`
	SelectedRemote string `json:"selected_remote"`
	RTTMillis      int64  `json:"rtt_ms"`
}

// GatherDirectCandidates 绑定 UDP socket 并收集 host/srflx 候选。
func GatherDirectCandidates(ctx context.Context, role string, stunServer string, logger *slog.Logger) (*Agent, CandidateExchange, error) {
	agentConfig := DefaultAgentConfig()
	agentConfig.STUNServer = stunServer
	agentConfig.Logger = logger
	agent := NewAgent(agentConfig)

	candidates, err := agent.GatherCandidates(ctx)
	if err != nil {
		_ = agent.Close()
		return nil, CandidateExchange{}, fmt.Errorf("gather candidates: %w", err)
	}

	sessionID, err := newVerifySessionID()
	if err != nil {
		_ = agent.Close()
		return nil, CandidateExchange{}, err
	}
	return agent, CandidateExchange{
		SessionID:  sessionID,
		Role:       role,
		Candidates: candidates,
	}, nil
}

// RunDirectConnectivity 完成远端候选注入、ICE 连通性检查和 UDP punch 验证。
func RunDirectConnectivity(ctx context.Context, role string, agent *Agent, local CandidateExchange, remote CandidateExchange, logger *slog.Logger) (DirectVerifyResult, error) {
	if agent == nil {
		return DirectVerifyResult{}, fmt.Errorf("agent is required")
	}
	sessionIDValue := directVerifySessionID(role, local, remote)
	if sessionIDValue == "" {
		return DirectVerifyResult{}, fmt.Errorf("direct verify session id is required")
	}
	for _, candidate := range remote.Candidates {
		agent.AddRemoteCandidate(candidate)
	}

	if err := agent.StartChecks(ctx); err != nil {
		return DirectVerifyResult{}, fmt.Errorf("ICE checks: %w", err)
	}
	pair := agent.GetSelectedPair()
	if pair == nil {
		return DirectVerifyResult{}, fmt.Errorf("no ICE pair selected")
	}

	agent.StopReadLoop()
	sessionID := verifySessionID(sessionIDValue)
	punchRole := PunchRoleInitiator
	if role == "responder" {
		punchRole = PunchRoleResponder
	}
	punch := NewPunchEngine(PunchConfig{
		SessionID:  sessionID,
		UDPConn:    agent.GetUDPConn(),
		RemoteAddr: &pair.Remote.Addr,
		Role:       punchRole,
		Timeout:    10 * time.Second,
		Logger:     logger,
	})
	result, err := punch.Punch(ctx)
	if err != nil {
		return DirectVerifyResult{}, fmt.Errorf("UDP punch: %w", err)
	}
	localAddr := ""
	if agent.GetUDPConn() != nil && agent.GetUDPConn().LocalAddr() != nil {
		localAddr = agent.GetUDPConn().LocalAddr().String()
	}
	return DirectVerifyResult{
		Role:           role,
		SessionID:      sessionIDValue,
		LocalAddr:      localAddr,
		RemoteAddr:     result.RemoteAddr.String(),
		SelectedLocal:  pair.Local.Addr.String(),
		SelectedRemote: pair.Remote.Addr.String(),
		RTTMillis:      result.RTT.Milliseconds(),
	}, nil
}

func directVerifySessionID(role string, local CandidateExchange, remote CandidateExchange) string {
	// 双端必须使用同一个打洞 session ID；responder 复用 initiator 发来的 ID。
	if role == "responder" {
		return remote.SessionID
	}
	return local.SessionID
}

func newVerifySessionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func verifySessionID(value string) [16]byte {
	decoded, _ := hex.DecodeString(value)
	var sessionID [16]byte
	copy(sessionID[:], decoded)
	return sessionID
}

// IsLANCandidate 用于报告里判断候选是否来自局域网直连地址。
func IsLANCandidate(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsPrivate()
}
