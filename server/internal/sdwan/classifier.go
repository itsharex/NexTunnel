package sdwan

import "sync"

// Classifier classifies network flows by application type based on port and protocol.
type Classifier struct {
	mu       sync.RWMutex
	portMap  map[int]AppType
}

// NewClassifier creates a classifier with default port mappings.
func NewClassifier() *Classifier {
	c := &Classifier{
		portMap: make(map[int]AppType),
	}
	// Default port-based classification
	c.portMap[80] = AppHTTP
	c.portMap[443] = AppHTTPS
	c.portMap[8080] = AppHTTP
	c.portMap[8443] = AppHTTPS
	c.portMap[22] = AppSSH
	c.portMap[3389] = AppRDP
	c.portMap[53] = AppDNS
	c.portMap[520] = AppWireGuard
	c.portMap[521] = AppWireGuard
	return c
}

// Classify determines the application type for a flow.
func (c *Classifier) Classify(flow FlowInfo) AppType {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check destination port first
	if app, ok := c.portMap[flow.DstPort]; ok {
		return app
	}

	// Check source port (for return traffic)
	if app, ok := c.portMap[flow.SrcPort]; ok {
		return app
	}

	// Protocol-based heuristics for UDP
	if flow.Protocol == ProtoUDP {
		return AppQUIC
	}

	return AppUnknown
}

// AddPortMapping adds a custom port-to-application mapping.
func (c *Classifier) AddPortMapping(port int, app AppType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.portMap[port] = app
}

// RemovePortMapping removes a port mapping.
func (c *Classifier) RemovePortMapping(port int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.portMap, port)
}

// MappingCount returns the number of port mappings.
func (c *Classifier) MappingCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.portMap)
}
