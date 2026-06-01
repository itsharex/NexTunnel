package nat

import (
	"context"
	"fmt"
	"log/slog"
	"net"
)

// Detector performs RFC 3489 Section 10.1 NAT type detection.
type Detector struct {
	serverAddr    string // primary STUN server address (ip:port)
	altServerAddr string // alternate STUN server address (different IP, same port)
	stunClient    STUNClient
	logger        *slog.Logger
}

// NewDetector creates a new NAT detector.
// serverAddr is the primary STUN server, altServerAddr is the alternate IP.
func NewDetector(serverAddr, altServerAddr string, stunClient STUNClient, logger *slog.Logger) *Detector {
	return &Detector{
		serverAddr:    serverAddr,
		altServerAddr: altServerAddr,
		stunClient:    stunClient,
		logger:        logger,
	}
}

// Detect runs the 4-test RFC 3489 NAT detection sequence and returns the NAT type.
func (d *Detector) Detect(ctx context.Context) (*NATResult, error) {
	d.logger.Info("starting NAT detection", "server", d.serverAddr, "alt", d.altServerAddr)

	// Bind a local UDP socket for tests
	localConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("bind local UDP: %w", err)
	}
	defer localConn.Close()

	localAddr := localConn.LocalAddr().(*net.UDPAddr)

	// Test I: Basic STUN binding from primary server
	binding1, err := d.stunClient.BindingRequest(ctx, d.serverAddr, localConn)
	if err != nil {
		d.logger.Warn("Test I failed: no STUN response", "error", err)
		return &NATResult{
			Type:      NATBlocked,
			LocalAddr: localAddr.String(),
		}, nil
	}

	mappedAddr1 := binding1.MappedAddr
	d.logger.Debug("Test I", "mapped", mappedAddr1.String(), "server", binding1.ResponseOrigin.String())

	// Check if local address matches mapped address (no NAT)
	if localAddr.IP.Equal(mappedAddr1.IP) && localAddr.Port == mappedAddr1.Port {
		// No NAT detected - check if open internet
		binding2, err := d.stunClient.BindingRequestFromAlt(ctx, d.serverAddr, localConn)
		if err == nil {
			_ = binding2
			d.logger.Info("NAT detection result: Open Internet")
			return &NATResult{
				Type:       NATOpenInternet,
				PublicAddr: mappedAddr1.String(),
				MappedPort: uint16(mappedAddr1.Port),
				LocalAddr:  localAddr.String(),
			}, nil
		}
		// Can't reach alternate server but no NAT
		return &NATResult{
			Type:       NATSymmetric, // firewall restricts alternate responses
			PublicAddr: mappedAddr1.String(),
			MappedPort: uint16(mappedAddr1.Port),
			LocalAddr:  localAddr.String(),
		}, nil
	}

	// We are behind NAT. Test II: ask server to respond from alternate IP:port
	binding2, err := d.stunClient.BindingRequestFromAlt(ctx, d.serverAddr, localConn)
	if err == nil {
		_ = binding2
		// Got response from alternate server -> Full Cone NAT
		d.logger.Info("NAT detection result: Full Cone")
		return &NATResult{
			Type:       NATFullCone,
			PublicAddr: mappedAddr1.String(),
			MappedPort: uint16(mappedAddr1.Port),
			LocalAddr:  localAddr.String(),
		}, nil
	}

	d.logger.Debug("Test II failed (no alternate response)", "error", err)

	// Test III: check if mapped address changes with different server address
	// Use a new local socket to test against the alternate server
	localConn2, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("bind second local UDP: %w", err)
	}
	defer localConn2.Close()

	binding3, err := d.stunClient.BindingRequest(ctx, d.altServerAddr, localConn2)
	if err != nil {
		d.logger.Warn("Test III failed: can't reach alternate server", "error", err)
		// Can't determine further, assume Port Restricted
		return &NATResult{
			Type:       NATPortRestricted,
			PublicAddr: mappedAddr1.String(),
			MappedPort: uint16(mappedAddr1.Port),
			LocalAddr:  localAddr.String(),
		}, nil
	}

	mappedAddr3 := binding3.MappedAddr
	d.logger.Debug("Test III", "mapped_alt", mappedAddr3.String())

	// Compare mapped addresses from two different servers
	if mappedAddr1.IP.Equal(mappedAddr3.IP) && mappedAddr1.Port == mappedAddr3.Port {
		// Same mapped address for different servers -> Restricted Cone NAT
		d.logger.Info("NAT detection result: Restricted Cone")
		return &NATResult{
			Type:       NATRestricted,
			PublicAddr: mappedAddr1.String(),
			MappedPort: uint16(mappedAddr1.Port),
			LocalAddr:  localAddr.String(),
		}, nil
	}

	// Different mapped address for different servers -> Symmetric NAT
	d.logger.Info("NAT detection result: Symmetric")
	return &NATResult{
		Type:       NATSymmetric,
		PublicAddr: mappedAddr1.String(),
		MappedPort: uint16(mappedAddr1.Port),
		LocalAddr:  localAddr.String(),
	}, nil
}
