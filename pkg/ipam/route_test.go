package ipam

import (
	"net"
	"testing"
)

func TestRouteManager_AddRoute(t *testing.T) {
	rm := NewRouteManager()

	_, subnet, _ := net.ParseCIDR("10.7.0.0/24")
	err := rm.AddRoute(Route{
		Destination: subnet,
		Gateway:     net.ParseIP("10.7.0.1"),
		Interface:   "tun0",
		Metric:      100,
	})
	if err != nil {
		t.Fatalf("AddRoute: %v", err)
	}

	routes := rm.ListRoutes()
	if len(routes) != 1 {
		t.Errorf("got %d routes, want 1", len(routes))
	}
}

func TestRouteManager_DuplicateRoute(t *testing.T) {
	rm := NewRouteManager()

	_, subnet, _ := net.ParseCIDR("10.7.0.0/24")
	rm.AddRoute(Route{Destination: subnet, Gateway: net.ParseIP("10.7.0.1")})

	err := rm.AddRoute(Route{Destination: subnet, Gateway: net.ParseIP("10.7.0.1")})
	if err == nil {
		t.Error("expected duplicate error")
	}
}

func TestRouteManager_RemoveRoute(t *testing.T) {
	rm := NewRouteManager()

	_, subnet, _ := net.ParseCIDR("10.7.0.0/24")
	rm.AddRoute(Route{Destination: subnet, Gateway: net.ParseIP("10.7.0.1")})

	err := rm.RemoveRoute(subnet)
	if err != nil {
		t.Fatalf("RemoveRoute: %v", err)
	}

	routes := rm.ListRoutes()
	if len(routes) != 0 {
		t.Errorf("got %d routes, want 0", len(routes))
	}
}

func TestRouteManager_RemoveNonexistent(t *testing.T) {
	rm := NewRouteManager()

	_, subnet, _ := net.ParseCIDR("10.7.0.0/24")
	err := rm.RemoveRoute(subnet)
	if err == nil {
		t.Error("expected error for nonexistent route")
	}
}

func TestRouteManager_FindRoute(t *testing.T) {
	rm := NewRouteManager()

	_, subnet1, _ := net.ParseCIDR("10.7.0.0/24")
	_, subnet2, _ := net.ParseCIDR("10.7.0.0/16")

	rm.AddRoute(Route{Destination: subnet1, Gateway: net.ParseIP("10.7.0.1"), Metric: 100})
	rm.AddRoute(Route{Destination: subnet2, Gateway: net.ParseIP("10.7.0.254"), Metric: 200})

	// 10.7.0.50 should match the more specific /24
	route := rm.FindRoute(net.ParseIP("10.7.0.50"))
	if route == nil {
		t.Fatal("no route found")
	}
	if route.Destination.String() != "10.7.0.0/24" {
		t.Errorf("expected /24, got %s", route.Destination)
	}

	// 10.7.1.50 should match the /16
	route = rm.FindRoute(net.ParseIP("10.7.1.50"))
	if route == nil {
		t.Fatal("no route found")
	}
	if route.Destination.String() != "10.7.0.0/16" {
		t.Errorf("expected /16, got %s", route.Destination)
	}

	// 192.168.1.1 should not match any route
	route = rm.FindRoute(net.ParseIP("192.168.1.1"))
	if route != nil {
		t.Error("should not find route for 192.168.1.1")
	}
}

func TestRoute_String(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("10.7.0.0/24")
	r := Route{Destination: subnet, Gateway: net.ParseIP("10.7.0.1"), Interface: "tun0", Metric: 100}
	s := r.String()
	if s == "" {
		t.Error("route string should not be empty")
	}
}
