package anycast

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// NodeInfo represents an edge node's routing information.
type NodeInfo struct {
	ID       string  `json:"id"`
	Addr     string  `json:"addr"`
	Region   string  `json:"region"`
	Lat      float64 `json:"lat"` // latitude
	Lon      float64 `json:"lon"` // longitude
	Healthy  bool    `json:"healthy"`
	Latency  time.Duration `json:"latency"`
	Priority int     `json:"priority"` // lower is better
}

// Router selects the nearest healthy edge node for a given client location.
type Router struct {
	config AnycastConfig
	mu     sync.RWMutex
	nodes  map[string]*NodeInfo
}

// NewRouter creates a new anycast router.
func NewRouter(cfg AnycastConfig) *Router {
	return &Router{
		config: cfg,
		nodes:  make(map[string]*NodeInfo),
	}
}

// AddNode registers a node for routing.
func (r *Router) AddNode(node *NodeInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[node.ID] = node
}

// RemoveNode removes a node from routing.
func (r *Router) RemoveNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nodes, nodeID)
}

// UpdateHealth updates a node's health status.
func (r *Router) UpdateHealth(nodeID string, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[nodeID]; ok {
		n.Healthy = healthy
	}
}

// SelectNearest returns the nearest healthy node to the given client coordinates.
// Returns nil if no healthy nodes are available.
func (r *Router) SelectNearest(clientLat, clientLon float64) *NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	candidates := r.healthyNodes()
	if len(candidates) == 0 {
		return nil
	}

	// Sort by haversine distance
	sort.Slice(candidates, func(i, j int) bool {
		di := haversine(clientLat, clientLon, candidates[i].Lat, candidates[i].Lon)
		dj := haversine(clientLat, clientLon, candidates[j].Lat, candidates[j].Lon)
		if di == dj {
			return candidates[i].Priority < candidates[j].Priority
		}
		return di < dj
	})

	return candidates[0]
}

// SelectNearestWithFailover returns up to `count` nearest healthy nodes for failover.
func (r *Router) SelectNearestWithFailover(clientLat, clientLon float64, count int) []*NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	candidates := r.healthyNodes()
	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		di := haversine(clientLat, clientLon, candidates[i].Lat, candidates[i].Lon)
		dj := haversine(clientLat, clientLon, candidates[j].Lat, candidates[j].Lon)
		if di == dj {
			return candidates[i].Priority < candidates[j].Priority
		}
		return di < dj
	})

	if count > len(candidates) {
		count = len(candidates)
	}
	return candidates[:count]
}

// SelectByRegion returns the nearest healthy node in a specific region.
func (r *Router) SelectByRegion(clientLat, clientLon float64, region string) *NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []*NodeInfo
	for _, n := range r.nodes {
		if n.Healthy && n.Region == region {
			candidates = append(candidates, n)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		di := haversine(clientLat, clientLon, candidates[i].Lat, candidates[i].Lon)
		dj := haversine(clientLat, clientLon, candidates[j].Lat, candidates[j].Lon)
		return di < dj
	})

	return candidates[0]
}

// NodeCount returns total registered nodes.
func (r *Router) NodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// HealthyCount returns the number of healthy nodes.
func (r *Router) HealthyCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.healthyNodes())
}

func (r *Router) healthyNodes() []*NodeInfo {
	var result []*NodeInfo
	for _, n := range r.nodes {
		if n.Healthy {
			result = append(result, n)
		}
	}
	return result
}

// haversine calculates the great-circle distance between two points in km.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth's radius in km
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// ResolveClientRegion maps an IP address to a geographic region.
// This is a simplified implementation; production would use a GeoIP database.
type GeoResolver struct {
	mu      sync.RWMutex
	mapping map[string]string // IP prefix -> region
}

// NewGeoResolver creates a new geo resolver with predefined mappings.
func NewGeoResolver() *GeoResolver {
	return &GeoResolver{
		mapping: make(map[string]string),
	}
}

// AddMapping adds an IP prefix to region mapping.
func (g *GeoResolver) AddMapping(ipPrefix, region string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.mapping[ipPrefix] = region
}

// Resolve returns the region for a given client IP.
func (g *GeoResolver) Resolve(clientIP string) (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check exact match first
	if region, ok := g.mapping[clientIP]; ok {
		return region, nil
	}

	// Check prefix match (simplified)
	for prefix, region := range g.mapping {
		if len(prefix) > 0 && len(clientIP) >= len(prefix) && clientIP[:len(prefix)] == prefix {
			return region, nil
		}
	}

	return "", fmt.Errorf("unknown region for IP: %s", clientIP)
}

// DNSRecord represents a GeoDNS response record.
type DNSRecord struct {
	Name    string   `json:"name"`
	Addrs   []string `json:"addrs"`
	TTL     int      `json:"ttl"`
	Region  string   `json:"region"`
}

// GeoDNsserver resolves DNS queries based on client geographic location.
type GeoDNSServer struct {
	config   AnycastConfig
	router   *Router
	geo      *GeoResolver
	domain   string
}

// NewGeoDNSServer creates a new GeoDNS server.
func NewGeoDNSServer(cfg AnycastConfig, router *Router, geo *GeoResolver, domain string) *GeoDNSServer {
	return &GeoDNSServer{
		config: cfg,
		router: router,
		geo:    geo,
		domain: domain,
	}
}

// Resolve handles a DNS query from a client IP, returning the nearest node address.
func (s *GeoDNSServer) Resolve(clientIP string) (*DNSRecord, error) {
	region, err := s.geo.Resolve(clientIP)
	if err != nil {
		// Fallback: return any healthy node
		node := s.router.SelectNearest(0, 0) // no location info, return first healthy
		if node == nil {
			return nil, fmt.Errorf("no healthy nodes available")
		}
		return &DNSRecord{
			Name:   s.domain,
			Addrs:  []string{node.Addr},
			TTL:    int(s.config.CacheTTL.Seconds()),
			Region: node.Region,
		}, nil
	}

	// Get region-specific coordinates (simplified)
	lat, lon := regionCoordinates(region)
	node := s.router.SelectNearest(lat, lon)
	if node == nil {
		return nil, fmt.Errorf("no healthy nodes for region %q", region)
	}

	return &DNSRecord{
		Name:   s.domain,
		Addrs:  []string{node.Addr},
		TTL:    int(s.config.CacheTTL.Seconds()),
		Region: region,
	}, nil
}

// regionCoordinates returns approximate coordinates for a region.
func regionCoordinates(region string) (float64, float64) {
	coords := map[string][2]float64{
		"us-east":       {39.0, -77.0},
		"us-west":       {37.0, -122.0},
		"eu-west":       {51.5, -0.1},
		"eu-central":    {50.1, 8.7},
		"ap-northeast":  {35.7, 139.7},
		"ap-southeast":  {1.3, 103.8},
		"ap-south":      {19.0, 72.8},
		"sa-east":       {-23.5, -46.6},
	}
	if c, ok := coords[region]; ok {
		return c[0], c[1]
	}
	return 0, 0
}
