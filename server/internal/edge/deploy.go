package edge

import (
	"fmt"
	"strings"
	"text/template"
)

// DeployConfig holds parameters for deploying a new edge node.
type DeployConfig struct {
	NodeID      string `json:"node_id"`
	Addr        string `json:"addr"`
	Region      string `json:"region"`
	Role        string `json:"role"`
	RelayPort   int    `json:"relay_port"`
	ControlAddr string `json:"control_addr"` // Control Plane address for registration
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
}

// DeployTemplate is the Docker Compose template for an edge node.
const DeployTemplate = `version: "3.8"

services:
  nexedge-{{.NodeID}}:
    image: nextunnel/relay-server:latest
    container_name: nexedge-{{.NodeID}}
    restart: unless-stopped
    ports:
      - "{{.RelayPort}}:{{.RelayPort}}/udp"
      - "{{.RelayPort}}:{{.RelayPort}}/tcp"
    environment:
      - NEXTUNNEL_NODE_ID={{.NodeID}}
      - NEXTUNNEL_REGION={{.Region}}
      - NEXTUNNEL_ROLE={{.Role}}
      - NEXTUNNEL_LISTEN_ADDR=0.0.0.0:{{.RelayPort}}
      - NEXTUNNEL_CONTROL_ADDR={{.ControlAddr}}
    {{- if .CertPath}}
      - NEXTUNNEL_CERT_PATH={{.CertPath}}
    {{- end}}
    {{- if .KeyPath}}
      - NEXTUNNEL_KEY_PATH={{.KeyPath}}
    {{- end}}
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "{{.RelayPort}}"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s
`

// GenerateDeployCompose generates a Docker Compose YAML for deploying an edge node.
func GenerateDeployCompose(cfg DeployConfig) (string, error) {
	if cfg.NodeID == "" {
		return "", fmt.Errorf("node_id is required")
	}
	if cfg.Region == "" {
		return "", fmt.Errorf("region is required")
	}
	if cfg.RelayPort == 0 {
		cfg.RelayPort = 4433
	}
	if cfg.Role == "" {
		cfg.Role = "full"
	}

	tmpl, err := template.New("deploy").Parse(DeployTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// DeployManifest describes a deployment result.
type DeployManifest struct {
	NodeID       string `json:"node_id"`
	ComposeFile  string `json:"compose_file"`
	Region       string `json:"region"`
	Instructions string `json:"instructions"`
}

// CreateDeployManifest generates a deployment manifest with instructions.
func CreateDeployManifest(cfg DeployConfig) (*DeployManifest, error) {
	compose, err := GenerateDeployCompose(cfg)
	if err != nil {
		return nil, err
	}

	return &DeployManifest{
		NodeID:      cfg.NodeID,
		ComposeFile: compose,
		Region:      cfg.Region,
		Instructions: fmt.Sprintf(
			"Deploy edge node %q in region %q:\n"+
				"1. Save compose content to docker-compose.yml\n"+
				"2. Run: docker compose up -d\n"+
				"3. Node will auto-register to Control Plane at %s\n"+
				"4. Verify: docker compose logs -f nexedge-%s",
			cfg.NodeID, cfg.Region, cfg.ControlAddr, cfg.NodeID,
		),
	}, nil
}
