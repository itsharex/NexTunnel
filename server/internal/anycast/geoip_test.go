package anycast

import (
	"testing"
)

func TestMaxMindAdapter_ExactMatch(t *testing.T) {
	adapter, err := NewMaxMindAdapter(MaxMindConfig{
		StaticMappings: map[string]*GeoIPResult{
			"203.0.113.1": {Region: "ap-northeast", Country: "JP", Latitude: 35.7, Longitude: 139.7},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	result, err := adapter.Lookup("203.0.113.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Region != "ap-northeast" {
		t.Fatalf("expected ap-northeast, got %s", result.Region)
	}
	if result.Country != "JP" {
		t.Fatalf("expected JP, got %s", result.Country)
	}
}

func TestMaxMindAdapter_CIDRMatch(t *testing.T) {
	adapter, _ := NewMaxMindAdapter(MaxMindConfig{
		StaticMappings: map[string]*GeoIPResult{
			"198.51.100.0/24": {Region: "eu-west", Country: "GB", Latitude: 51.5, Longitude: -0.1},
		},
	})
	defer adapter.Close()

	result, err := adapter.Lookup("198.51.100.42")
	if err != nil {
		t.Fatal(err)
	}
	if result.Region != "eu-west" {
		t.Fatalf("expected eu-west, got %s", result.Region)
	}
}

func TestMaxMindAdapter_Default(t *testing.T) {
	adapter, _ := NewMaxMindAdapter(MaxMindConfig{
		DefaultResult: &GeoIPResult{Region: "us-east", Country: "US"},
	})
	defer adapter.Close()

	result, err := adapter.Lookup("1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if result.Region != "us-east" {
		t.Fatalf("expected us-east default, got %s", result.Region)
	}
}

func TestMaxMindAdapter_NotFound(t *testing.T) {
	adapter, _ := NewMaxMindAdapter(MaxMindConfig{})
	defer adapter.Close()

	_, err := adapter.Lookup("1.2.3.4")
	if err == nil {
		t.Fatal("expected error for unknown IP")
	}
}

func TestMaxMindAdapter_InvalidIP(t *testing.T) {
	adapter, _ := NewMaxMindAdapter(MaxMindConfig{})
	defer adapter.Close()

	_, err := adapter.Lookup("not-an-ip")
	if err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

func TestMaxMindAdapter_AddRemoveMapping(t *testing.T) {
	adapter, _ := NewMaxMindAdapter(MaxMindConfig{})
	defer adapter.Close()

	adapter.AddMapping("10.0.0.1", &GeoIPResult{Region: "ap-south"})
	if adapter.MappingCount() != 1 {
		t.Fatalf("expected 1 mapping, got %d", adapter.MappingCount())
	}

	result, err := adapter.Lookup("10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Region != "ap-south" {
		t.Fatalf("expected ap-south, got %s", result.Region)
	}

	adapter.RemoveMapping("10.0.0.1")
	if adapter.MappingCount() != 0 {
		t.Fatalf("expected 0 mappings, got %d", adapter.MappingCount())
	}
}

func TestStaticGeoProvider_Lookup(t *testing.T) {
	resolver := NewGeoResolver()
	resolver.AddMapping("10.0.0", "eu-central")

	provider := NewStaticGeoProvider(resolver)
	defer provider.Close()

	result, err := provider.Lookup("10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Region != "eu-central" {
		t.Fatalf("expected eu-central, got %s", result.Region)
	}
	if result.Latitude != 50.1 {
		t.Fatalf("expected lat 50.1, got %f", result.Latitude)
	}
}

func TestStaticGeoProvider_NotFound(t *testing.T) {
	resolver := NewGeoResolver()
	provider := NewStaticGeoProvider(resolver)

	_, err := provider.Lookup("99.99.99.99")
	if err == nil {
		t.Fatal("expected error for unknown IP")
	}
}

func TestGeoIPRouter_SelectNearestForIP(t *testing.T) {
	cfg := DefaultAnycastConfig()
	router := NewRouter(cfg)

	// Add nodes
	router.AddNode(&NodeInfo{ID: "n1", Addr: "1.1.1.1:443", Region: "us-east", Lat: 39.0, Lon: -77.0, Healthy: true})
	router.AddNode(&NodeInfo{ID: "n2", Addr: "2.2.2.2:443", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})
	router.AddNode(&NodeInfo{ID: "n3", Addr: "3.3.3.3:443", Region: "ap-northeast", Lat: 35.7, Lon: 139.7, Healthy: true})

	adapter, _ := NewMaxMindAdapter(MaxMindConfig{
		StaticMappings: map[string]*GeoIPResult{
			"203.0.113.50": {Region: "ap-northeast", Latitude: 35.0, Longitude: 139.0},
		},
	})

	geoRouter := NewGeoIPRouter(router, adapter)

	// Client in Japan should get ap-northeast node
	node := geoRouter.SelectNearestForIP("203.0.113.50")
	if node == nil {
		t.Fatal("expected a node")
	}
	if node.ID != "n3" {
		t.Fatalf("expected n3 (ap-northeast), got %s (%s)", node.ID, node.Region)
	}
}

func TestGeoIPRouter_SelectNearestForIP_UnknownIP(t *testing.T) {
	cfg := DefaultAnycastConfig()
	router := NewRouter(cfg)
	router.AddNode(&NodeInfo{ID: "n1", Addr: "1.1.1.1:443", Healthy: true})

	adapter, _ := NewMaxMindAdapter(MaxMindConfig{})
	geoRouter := NewGeoIPRouter(router, adapter)

	// Unknown IP should fall back to any healthy node
	node := geoRouter.SelectNearestForIP("99.99.99.99")
	if node == nil {
		t.Fatal("expected fallback node")
	}
}

func TestGeoIPRouter_FailoverForIP(t *testing.T) {
	cfg := DefaultAnycastConfig()
	router := NewRouter(cfg)

	router.AddNode(&NodeInfo{ID: "n1", Addr: "1.1.1.1:443", Region: "us-east", Lat: 39.0, Lon: -77.0, Healthy: true})
	router.AddNode(&NodeInfo{ID: "n2", Addr: "2.2.2.2:443", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})

	adapter, _ := NewMaxMindAdapter(MaxMindConfig{
		StaticMappings: map[string]*GeoIPResult{
			"10.0.0.1": {Region: "us-east", Latitude: 40.0, Longitude: -74.0},
		},
	})

	geoRouter := NewGeoIPRouter(router, adapter)

	nodes := geoRouter.SelectNearestWithFailoverForIP("10.0.0.1", 2)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 failover nodes, got %d", len(nodes))
	}
	// First should be us-east (closest)
	if nodes[0].Region != "us-east" {
		t.Fatalf("expected first node us-east, got %s", nodes[0].Region)
	}
}

func TestGeoIPRouter_SelectByRegionForIP(t *testing.T) {
	cfg := DefaultAnycastConfig()
	router := NewRouter(cfg)

	router.AddNode(&NodeInfo{ID: "n1", Addr: "1.1.1.1:443", Region: "us-east", Lat: 39.0, Lon: -77.0, Healthy: true})
	router.AddNode(&NodeInfo{ID: "n2", Addr: "2.2.2.2:443", Region: "eu-west", Lat: 51.5, Lon: -0.1, Healthy: true})

	adapter, _ := NewMaxMindAdapter(MaxMindConfig{
		StaticMappings: map[string]*GeoIPResult{
			"10.0.0.1": {Region: "eu-west", Latitude: 52.0, Longitude: 0.0},
		},
	})

	geoRouter := NewGeoIPRouter(router, adapter)

	node := geoRouter.SelectByRegionForIP("10.0.0.1", "eu-west")
	if node == nil {
		t.Fatal("expected a node")
	}
	if node.ID != "n2" {
		t.Fatalf("expected n2, got %s", node.ID)
	}
}
