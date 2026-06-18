package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/nextunnel/server/internal/anycast"
	"github.com/nextunnel/server/internal/edge"
)

const (
	defaultHeartbeatInterval = 2 * time.Second
	defaultHTTPTimeout       = 10 * time.Second
)

type checkResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type rehearsalReport struct {
	GeneratedAt string        `json:"generated_at"`
	Passed      bool          `json:"passed"`
	ControlURL  string        `json:"control_url,omitempty"`
	Checks      []checkResult `json:"checks"`
}

func main() {
	var controlURL string
	var controlToken string
	var registerRemote bool
	var heartbeatWait time.Duration

	flag.StringVar(&controlURL, "control-url", "", "optional Control Plane base URL for real registration checks")
	flag.StringVar(&controlToken, "control-token", "", "optional Control Plane Bearer token")
	flag.BoolVar(&registerRemote, "register-remote", false, "register and delete rehearsal nodes against the real Control Plane")
	flag.DurationVar(&heartbeatWait, "heartbeat-wait", 3*time.Second, "time to wait for heartbeat loop during remote registration")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	report := rehearsalReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		ControlURL:  controlURL,
		Checks:      make([]checkResult, 0, 8),
	}

	nodes := mustCreateNodes()
	registry := edge.NewRegistry()
	for _, node := range nodes {
		if err := registry.Register(node); err != nil {
			report.add("edge_registry_register", false, err.Error())
			writeReportAndExit(report)
		}
	}
	report.add("edge_registry_register", registry.CountHealthy() == len(nodes), fmt.Sprintf("healthy=%d total=%d", registry.CountHealthy(), len(nodes)))

	router := anycast.NewRouter(anycast.DefaultAnycastConfig())
	router.AddNode(&anycast.NodeInfo{ID: "edge-us", Addr: "198.51.100.10:7000", Region: "us-east", Lat: 39.0, Lon: -77.0, Healthy: true, Priority: 10})
	router.AddNode(&anycast.NodeInfo{ID: "edge-eu", Addr: "198.51.100.20:7000", Region: "eu-central", Lat: 50.1, Lon: 8.7, Healthy: true, Priority: 10})
	router.AddNode(&anycast.NodeInfo{ID: "edge-ap", Addr: "198.51.100.30:7000", Region: "ap-southeast", Lat: 1.3, Lon: 103.8, Healthy: true, Priority: 10})

	usNearest := router.SelectNearest(40.7, -74.0)
	report.add("anycast_nearest_us", usNearest != nil && usNearest.ID == "edge-us", nodeDetail(usNearest))

	router.UpdateHealth("edge-us", false)
	failover := router.SelectNearestWithFailover(40.7, -74.0, 2)
	report.add("anycast_failover", len(failover) > 0 && failover[0].ID != "edge-us", failoverDetail(failover))

	provider, err := anycast.NewMaxMindAdapter(anycast.MaxMindConfig{
		StaticMappings: map[string]*anycast.GeoIPResult{
			"203.0.113.0/24":  {Region: "ap-southeast", Country: "SG", City: "Singapore", Latitude: 1.3, Longitude: 103.8},
			"198.51.100.0/24": {Region: "us-east", Country: "US", City: "Virginia", Latitude: 39.0, Longitude: -77.0},
		},
	})
	if err != nil {
		report.add("geoip_provider", false, err.Error())
		writeReportAndExit(report)
	}
	defer provider.Close()

	geoRouter := anycast.NewGeoIPRouter(router, provider)
	apNearest := geoRouter.SelectNearestForIP("203.0.113.42")
	report.add("geoip_route_shift", apNearest != nil && apNearest.ID == "edge-ap", nodeDetail(apNearest))

	if registerRemote {
		if controlURL == "" {
			report.add("control_plane_remote_registration", false, "--control-url is required when --register-remote is set")
			writeReportAndExit(report)
		}
		client := edge.NewControlPlaneClient(edge.ControlPlaneConfig{
			BaseURL:           controlURL,
			BearerToken:       controlToken,
			HeartbeatInterval: defaultHeartbeatInterval,
			Timeout:           defaultHTTPTimeout,
			Logger:            logger,
		})
		for _, node := range nodes {
			if err := client.RegisterNode(context.Background(), node); err != nil {
				report.add("control_plane_register_"+node.ID, false, err.Error())
				writeReportAndExit(report)
			}
			client.StartHeartbeatLoop(node)
		}
		time.Sleep(heartbeatWait)
		for _, node := range nodes {
			client.StopHeartbeatLoop(node.ID)
			if err := client.SendHeartbeat(context.Background(), node); err != nil {
				report.add("control_plane_heartbeat_"+node.ID, false, err.Error())
				writeReportAndExit(report)
			}
		}
		client.StopAll()
		for _, node := range nodes {
			if err := deleteControlPlaneNode(controlURL, controlToken, node.ID); err != nil {
				report.add("control_plane_cleanup_"+node.ID, false, err.Error())
				writeReportAndExit(report)
			}
		}
		report.add("control_plane_remote_registration", true, fmt.Sprintf("registered=%d heartbeat_wait=%s", len(nodes), heartbeatWait))
	}

	writeReportAndExit(report)
}

func mustCreateNodes() []*edge.EdgeNode {
	definitions := []struct {
		id     string
		addr   string
		region string
	}{
		{"edge-us", "127.0.0.1:17001", "us-east"},
		{"edge-eu", "127.0.0.1:17002", "eu-central"},
		{"edge-ap", "127.0.0.1:17003", "ap-southeast"},
	}
	nodes := make([]*edge.EdgeNode, 0, len(definitions))
	for _, definition := range definitions {
		node, err := edge.NewEdgeNode(definition.id, definition.addr, definition.region, edge.RoleFull, 1000)
		if err != nil {
			panic(err)
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func nodeDetail(node *anycast.NodeInfo) string {
	if node == nil {
		return "node=nil"
	}
	return fmt.Sprintf("id=%s region=%s addr=%s", node.ID, node.Region, node.Addr)
}

func failoverDetail(nodes []*anycast.NodeInfo) string {
	if len(nodes) == 0 {
		return "none"
	}
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	return fmt.Sprintf("%v", ids)
}

func deleteControlPlaneNode(baseURL, token, nodeID string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/nodes/%s", baseURL, nodeID), nil)
	if err != nil {
		return fmt.Errorf("create cleanup request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: defaultHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete node %s: %w", nodeID, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete node %s returned HTTP %d", nodeID, resp.StatusCode)
	}
	return nil
}

func (r *rehearsalReport) add(name string, passed bool, detail string) {
	r.Checks = append(r.Checks, checkResult{Name: name, Passed: passed, Detail: detail})
}

func (r *rehearsalReport) finalize() {
	r.Passed = true
	for _, check := range r.Checks {
		if !check.Passed {
			r.Passed = false
			return
		}
	}
}

func writeReportAndExit(report rehearsalReport) {
	report.finalize()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	if !report.Passed {
		os.Exit(1)
	}
}
