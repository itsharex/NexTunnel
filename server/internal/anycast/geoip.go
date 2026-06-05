package anycast

import (
	"fmt"
	"net"
	"sync"
)

// GeoIPResult holds the result of a GeoIP lookup.
type GeoIPResult struct {
	Region    string  `json:"region"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GeoIPProvider is the interface for IP-to-geolocation resolution.
// Implementations can use MaxMind GeoLite2/GeoIP2 databases, IP2Location,
// or any other GeoIP data source.
type GeoIPProvider interface {
	// Lookup resolves an IP address to geographic information.
	Lookup(ip string) (*GeoIPResult, error)

	// Close releases any resources held by the provider.
	Close() error
}

// MaxMindAdapter adapts a MaxMind GeoIP2/GeoLite2 database to the GeoIPProvider interface.
// In production, this would use github.com/oschwald/geoip2-golang to read the .mmdb file.
// The current implementation provides a configurable mapping for testing and development.
type MaxMindAdapter struct {
	mu       sync.RWMutex
	dbPath   string
	mappings map[string]*GeoIPResult // IP prefix -> result
	defaults *GeoIPResult            // fallback for unmatched IPs
}

// MaxMindConfig configures the MaxMind adapter.
type MaxMindConfig struct {
	// DBPath is the path to the .mmdb database file.
	// If empty, the adapter operates in mapping-only mode (for testing).
	DBPath string

	// StaticMappings provides predefined IP-to-location mappings.
	// Keys can be exact IPs or CIDR prefixes (e.g. "203.0.113.0/24").
	StaticMappings map[string]*GeoIPResult

	// DefaultResult is returned when no mapping matches and no DB is loaded.
	DefaultResult *GeoIPResult
}

// NewMaxMindAdapter creates a new MaxMind adapter.
// In production, this would open the .mmdb file using geoip2.Open(dbPath).
func NewMaxMindAdapter(cfg MaxMindConfig) (*MaxMindAdapter, error) {
	a := &MaxMindAdapter{
		dbPath:   cfg.DBPath,
		mappings: make(map[string]*GeoIPResult),
		defaults: cfg.DefaultResult,
	}

	for prefix, result := range cfg.StaticMappings {
		a.mappings[prefix] = result
	}

	// In production: if cfg.DBPath != "", open the MaxMind database:
	//   db, err := geoip2.Open(cfg.DBPath)
	//   if err != nil { return nil, fmt.Errorf("open maxmind db: %w", err) }
	//   a.db = db
	//
	// For now, we log the intent and operate with static mappings.

	return a, nil
}

// Lookup resolves an IP address using the MaxMind database or static mappings.
func (a *MaxMindAdapter) Lookup(ip string) (*GeoIPResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 1. Check exact match
	if result, ok := a.mappings[ip]; ok {
		return result, nil
	}

	// 2. Check CIDR prefix match
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	for prefix, result := range a.mappings {
		_, cidr, err := net.ParseCIDR(prefix)
		if err != nil {
			continue // not a CIDR, skip
		}
		if cidr.Contains(parsedIP) {
			return result, nil
		}
	}

	// 3. In production with a loaded DB:
	//   record, err := a.db.City(parsedIP)
	//   if err != nil { return nil, err }
	//   return &GeoIPResult{
	//       Region:    record.Subdivisions[0].IsoCode,
	//       Country:   record.Country.IsoCode,
	//       City:      record.City.Names["en"],
	//       Latitude:  record.Location.Latitude,
	//       Longitude: record.Location.Longitude,
	//   }, nil

	// 4. Return default if available
	if a.defaults != nil {
		return a.defaults, nil
	}

	return nil, fmt.Errorf("no GeoIP data for IP: %s", ip)
}

// Close releases the MaxMind database resources.
func (a *MaxMindAdapter) Close() error {
	// In production: a.db.Close()
	return nil
}

// AddMapping adds a runtime IP-to-location mapping.
func (a *MaxMindAdapter) AddMapping(prefix string, result *GeoIPResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mappings[prefix] = result
}

// RemoveMapping removes a mapping.
func (a *MaxMindAdapter) RemoveMapping(prefix string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.mappings, prefix)
}

// MappingCount returns the number of static mappings.
func (a *MaxMindAdapter) MappingCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.mappings)
}

// StaticGeoProvider wraps the existing GeoResolver as a GeoIPProvider.
// This enables backward compatibility while the codebase migrates to the interface.
type StaticGeoProvider struct {
	resolver *GeoResolver
	coords   map[string][2]float64 // region -> [lat, lon]
}

// NewStaticGeoProvider creates a provider from the legacy GeoResolver.
func NewStaticGeoProvider(resolver *GeoResolver) *StaticGeoProvider {
	return &StaticGeoProvider{
		resolver: resolver,
		coords: map[string][2]float64{
			"us-east":      {39.0, -77.0},
			"us-west":      {37.0, -122.0},
			"eu-west":      {51.5, -0.1},
			"eu-central":   {50.1, 8.7},
			"ap-northeast": {35.7, 139.7},
			"ap-southeast": {1.3, 103.8},
			"ap-south":     {19.0, 72.8},
			"sa-east":      {-23.5, -46.6},
		},
	}
}

// Lookup resolves an IP using the legacy GeoResolver.
func (p *StaticGeoProvider) Lookup(ip string) (*GeoIPResult, error) {
	region, err := p.resolver.Resolve(ip)
	if err != nil {
		return nil, err
	}
	coords, ok := p.coords[region]
	if !ok {
		return &GeoIPResult{Region: region}, nil
	}
	return &GeoIPResult{
		Region:    region,
		Latitude:  coords[0],
		Longitude: coords[1],
	}, nil
}

// Close is a no-op for the static provider.
func (p *StaticGeoProvider) Close() error { return nil }

// GeoIPRouter wraps a Router with a GeoIPProvider for enhanced resolution.
type GeoIPRouter struct {
	router   *Router
	provider GeoIPProvider
}

// NewGeoIPRouter creates a GeoIP-aware router.
func NewGeoIPRouter(router *Router, provider GeoIPProvider) *GeoIPRouter {
	return &GeoIPRouter{
		router:   router,
		provider: provider,
	}
}

// SelectNearestForIP finds the nearest healthy node for a client IP address.
func (g *GeoIPRouter) SelectNearestForIP(clientIP string) *NodeInfo {
	result, err := g.provider.Lookup(clientIP)
	if err != nil || result == nil {
		// Fallback: return any healthy node
		return g.router.SelectNearest(0, 0)
	}
	return g.router.SelectNearest(result.Latitude, result.Longitude)
}

// SelectNearestWithFailoverForIP returns up to count nearest nodes for a client IP.
func (g *GeoIPRouter) SelectNearestWithFailoverForIP(clientIP string, count int) []*NodeInfo {
	result, err := g.provider.Lookup(clientIP)
	if err != nil || result == nil {
		return g.router.SelectNearestWithFailover(0, 0, count)
	}
	return g.router.SelectNearestWithFailover(result.Latitude, result.Longitude, count)
}

// SelectByRegionForIP finds the nearest node in a specific region for a client IP.
func (g *GeoIPRouter) SelectByRegionForIP(clientIP, region string) *NodeInfo {
	result, err := g.provider.Lookup(clientIP)
	if err != nil || result == nil {
		return g.router.SelectByRegion(0, 0, region)
	}
	return g.router.SelectByRegion(result.Latitude, result.Longitude, region)
}
