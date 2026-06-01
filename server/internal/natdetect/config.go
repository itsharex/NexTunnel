package natdetect

import "time"

// Config holds the NAT detection server configuration.
type Config struct {
	PrimaryAddr string        `json:"primary_addr"`
	AltAddr     string        `json:"alt_addr"`
	Port        int           `json:"port"`
	Realm       string        `json:"realm"`
	AuthSecret  string        `json:"auth_secret"` // shared secret for TURN auth (optional)
	Timeout     time.Duration `json:"timeout"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		PrimaryAddr: "0.0.0.0",
		AltAddr:     "127.0.0.1",
		Port:        3478,
		Realm:       "nextunnel.local",
		Timeout:     5 * time.Second,
	}
}
