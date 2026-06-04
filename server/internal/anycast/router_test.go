package anycast

import (
	"math"
	"testing"
	"time"
)

func TestHaversine_Distance(t *testing.T) {
	// New York to London ≈ 5570 km
	d := haversine(40.7, -74.0, 51.5, -0.1)
	if math.Abs(d-5570) > 100 {
		t.Errorf("NYC-London distance = %.0f km, expected ~5570 km", d)
	}

	// Same point
	d0 := haversine(0, 0, 0, 0)
	if d0 != 0 {
		t.Errorf("same point distance = %f, want 0", d0)
	}

	t.Logf("NYC-London: %.0f km", d)
}

func TestRouter_SelectNearest(t *testing.T) {
	cfg := DefaultAnycastConfig()
	r := NewRouter(cfg)

	// Add nodes across the globe
	nodes := []*NodeInfo{
		{ID: "nyc", Addr: "1.1.1.1:4433", Region: "us-east", Lat: 40.7, Lon: -74.0, Healthy: true},
		{ID: "london", Addr: "2.2.2.2:4433", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true},
		{ID: "tokyo", Addr: "3.3.3.3:4433", Region: "ap-northeast", Lat: 35.7, Lon: 139.7, Healthy: true},
		{ID: "singapore", Addr: "4.4.4.4:4433", Region: "ap-southeast", Lat: 1.3, Lon: 103.8, Healthy: true},
	}

	for _, n := range nodes {
		r.AddNode(n)
	}

	if r.NodeCount() != 4 {
		t.Errorf("NodeCount = %d, want 4", r.NodeCount())
	}

	// Client in Paris (48.8, 2.3) → should select London
	nearest := r.SelectNearest(48.8, 2.3)
	if nearest == nil {
		t.Fatal("SelectNearest returned nil")
	}
	if nearest.ID != "london" {
		t.Errorf("Paris client → %s, want london", nearest.ID)
	}

	// Client in Seoul (37.5, 127.0) → should select Tokyo
	nearest = r.SelectNearest(37.5, 127.0)
	if nearest == nil {
		t.Fatal("SelectNearest returned nil")
	}
	if nearest.ID != "tokyo" {
		t.Errorf("Seoul client → %s, want tokyo", nearest.ID)
	}

	t.Logf("Paris → %s, Seoul → %s", "london", "tokyo")
}

func TestRouter_Failover(t *testing.T) {
	cfg := DefaultAnycastConfig()
	r := NewRouter(cfg)

	r.AddNode(&NodeInfo{ID: "primary", Addr: "1.1.1.1:4433", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})
	r.AddNode(&NodeInfo{ID: "backup", Addr: "2.2.2.2:4433", Region: "eu-central", Lat: 50.1, Lon: 8.7, Healthy: true})

	// Get failover list for Paris
	results := r.SelectNearestWithFailover(48.8, 2.3, 2)
	if len(results) != 2 {
		t.Fatalf("failover count = %d, want 2", len(results))
	}
	if results[0].ID != "primary" {
		t.Errorf("first choice = %s, want primary", results[0].ID)
	}
	if results[1].ID != "backup" {
		t.Errorf("second choice = %s, want backup", results[1].ID)
	}

	// Mark primary unhealthy → should fallback to backup
	r.UpdateHealth("primary", false)
	nearest := r.SelectNearest(48.8, 2.3)
	if nearest == nil || nearest.ID != "backup" {
		t.Errorf("after failover: got %v, want backup", nearest)
	}

	if r.HealthyCount() != 1 {
		t.Errorf("HealthyCount = %d, want 1", r.HealthyCount())
	}

	t.Logf("Failover: primary unhealthy, switched to %s", nearest.ID)
}

func TestRouter_NoHealthyNodes(t *testing.T) {
	r := NewRouter(DefaultAnycastConfig())
	r.AddNode(&NodeInfo{ID: "n1", Addr: "1.1.1.1:4433", Region: "us-east", Healthy: false})

	result := r.SelectNearest(40.0, -74.0)
	if result != nil {
		t.Errorf("expected nil when no healthy nodes, got %v", result)
	}
}

func TestRouter_SelectByRegion(t *testing.T) {
	r := NewRouter(DefaultAnycastConfig())
	r.AddNode(&NodeInfo{ID: "nyc", Addr: "1.1.1.1:4433", Region: "us-east", Lat: 40.7, Lon: -74.0, Healthy: true})
	r.AddNode(&NodeInfo{ID: "london", Addr: "2.2.2.2:4433", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})

	node := r.SelectByRegion(48.8, 2.3, "eu-west")
	if node == nil || node.ID != "london" {
		t.Errorf("SelectByRegion eu-west = %v, want london", node)
	}

	node = r.SelectByRegion(48.8, 2.3, "nonexistent")
	if node != nil {
		t.Errorf("SelectByRegion nonexistent = %v, want nil", node)
	}
}

func TestGeoResolver_Resolve(t *testing.T) {
	geo := NewGeoResolver()
	geo.AddMapping("10.0.", "us-east")
	geo.AddMapping("172.16.", "eu-west")
	geo.AddMapping("192.168.1.", "ap-northeast")

	tests := []struct {
		ip         string
		wantRegion string
		wantErr    bool
	}{
		{"10.0.1.5", "us-east", false},
		{"172.16.5.10", "eu-west", false},
		{"192.168.1.100", "ap-northeast", false},
		{"8.8.8.8", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			region, err := geo.Resolve(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve(%s) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
			if region != tt.wantRegion {
				t.Errorf("Resolve(%s) = %q, want %q", tt.ip, region, tt.wantRegion)
			}
		})
	}
}

func TestGeoDNSServer_Resolve(t *testing.T) {
	cfg := DefaultAnycastConfig()
	cfg.CacheTTL = 30 * time.Second

	router := NewRouter(cfg)
	router.AddNode(&NodeInfo{ID: "nyc", Addr: "1.1.1.1:4433", Region: "us-east", Lat: 40.7, Lon: -74.0, Healthy: true})
	router.AddNode(&NodeInfo{ID: "london", Addr: "2.2.2.2:4433", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})

	geo := NewGeoResolver()
	geo.AddMapping("10.0.", "us-east")
	geo.AddMapping("172.16.", "eu-west")

	dns := NewGeoDNSServer(cfg, router, geo, "relay.nextunnel.io")

	// US client
	record, err := dns.Resolve("10.0.1.5")
	if err != nil {
		t.Fatalf("Resolve US: %v", err)
	}
	if record.Addrs[0] != "1.1.1.1:4433" {
		t.Errorf("US client → %s, want 1.1.1.1:4433", record.Addrs[0])
	}
	if record.TTL != 30 {
		t.Errorf("TTL = %d, want 30", record.TTL)
	}

	// EU client
	record, err = dns.Resolve("172.16.5.10")
	if err != nil {
		t.Fatalf("Resolve EU: %v", err)
	}
	if record.Addrs[0] != "2.2.2.2:4433" {
		t.Errorf("EU client → %s, want 2.2.2.2:4433", record.Addrs[0])
	}

	// Unknown region (fallback)
	record, err = dns.Resolve("8.8.8.8")
	if err != nil {
		t.Fatalf("Resolve unknown: %v", err)
	}
	if len(record.Addrs) == 0 {
		t.Error("expected fallback address")
	}

	t.Logf("GeoDNS: US→%s EU→%s Unknown→%s", "1.1.1.1", "2.2.2.2", record.Addrs[0])
}

func TestRouter_RemoveNode(t *testing.T) {
	r := NewRouter(DefaultAnycastConfig())
	r.AddNode(&NodeInfo{ID: "n1", Addr: "1.1.1.1:4433", Region: "us-east", Lat: 40.7, Lon: -74.0, Healthy: true})
	r.AddNode(&NodeInfo{ID: "n2", Addr: "2.2.2.2:4433", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})

	if r.NodeCount() != 2 {
		t.Fatalf("NodeCount = %d, want 2", r.NodeCount())
	}

	r.RemoveNode("n1")
	if r.NodeCount() != 1 {
		t.Errorf("NodeCount after remove = %d, want 1", r.NodeCount())
	}

	nearest := r.SelectNearest(40.0, -74.0)
	if nearest == nil || nearest.ID != "n2" {
		t.Errorf("after remove n1: got %v, want n2", nearest)
	}
}
