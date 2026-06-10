package ipam

import (
	"fmt"
	"net"
	"sync"
)

// Route represents a network route entry.
type Route struct {
	Destination *net.IPNet
	Gateway     net.IP
	Interface   string
	Metric      int
}

// String returns a human-readable route description.
func (r Route) String() string {
	return fmt.Sprintf("%s via %s dev %s metric %d", r.Destination.String(), r.Gateway, r.Interface, r.Metric)
}

// RouteManager manages a table of network routes.
type RouteManager struct {
	mu     sync.RWMutex
	routes []Route
}

// NewRouteManager creates a new route manager.
func NewRouteManager() *RouteManager {
	return &RouteManager{}
}

// AddRoute adds a route to the routing table.
func (rm *RouteManager) AddRoute(route Route) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check for duplicate
	for _, r := range rm.routes {
		if r.Destination.String() == route.Destination.String() && r.Gateway.Equal(route.Gateway) {
			return fmt.Errorf("route %s already exists", route)
		}
	}

	rm.routes = append(rm.routes, route)
	return nil
}

// RemoveRoute removes a route matching the destination.
func (rm *RouteManager) RemoveRoute(destination *net.IPNet) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for i, r := range rm.routes {
		if r.Destination.String() == destination.String() {
			rm.routes = append(rm.routes[:i], rm.routes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("route %s not found", destination.String())
}

// ListRoutes returns all routes.
func (rm *RouteManager) ListRoutes() []Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	result := make([]Route, len(rm.routes))
	copy(result, rm.routes)
	return result
}

// FindRoute returns the best matching route for the given IP.
func (rm *RouteManager) FindRoute(ip net.IP) *Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var best *Route
	bestMask := 0
	for i, r := range rm.routes {
		if r.Destination.Contains(ip) {
			ones, _ := r.Destination.Mask.Size()
			if ones > bestMask {
				bestMask = ones
				best = &rm.routes[i]
			}
		}
	}
	return best
}
