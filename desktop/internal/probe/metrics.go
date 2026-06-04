package probe

import "time"

// LinkMetrics holds the current quality measurements for a network path.
type LinkMetrics struct {
	// PathID identifies the path (e.g., "udp_p2p", "quic_relay_1").
	PathID string `json:"path_id"`

	// RTT is the smoothed round-trip time (EWMA).
	RTT time.Duration `json:"rtt"`

	// RTTMin is the minimum observed RTT.
	RTTMin time.Duration `json:"rtt_min"`

	// RTTMax is the maximum observed RTT.
	RTTMax time.Duration `json:"rtt_max"`

	// RTTJitter is the RTT standard deviation.
	RTTJitter time.Duration `json:"rtt_jitter"`

	// LossRate is the packet loss ratio (0.0 to 1.0).
	LossRate float64 `json:"loss_rate"`

	// Bandwidth is the estimated bandwidth in bits per second.
	Bandwidth int64 `json:"bandwidth_bps"`

	// LastUpdated is the time of the last measurement.
	LastUpdated time.Time `json:"last_updated"`

	// SampleCount is the total number of probe samples collected.
	SampleCount int64 `json:"sample_count"`
}

// RTTSample holds a single RTT measurement.
type RTTSample struct {
	SendTime    time.Time
	ReceiveTime time.Time
	RTT         time.Duration
	SeqNum      uint16
	Acked       bool
}

// LossSample holds a loss measurement over a window.
type LossSample struct {
	Sent     uint64
	Received uint64
	LossRate float64
	Window   time.Duration
}

// BandwidthSample holds a bandwidth estimation sample.
type BandwidthSample struct {
	Bytes     int64
	Duration  time.Duration
	Bandwidth int64 // bits per second
}

// IsReachable returns true if the path appears to be reachable.
func (m *LinkMetrics) IsReachable() bool {
	return m.RTT > 0 && m.RTT < 30*time.Second
}

// QualityScore returns a composite quality score (0.0-1.0, higher is better).
func (m *LinkMetrics) QualityScore() float64 {
	if !m.IsReachable() {
		return 0
	}

	rttScore := 1.0 / (1.0 + float64(m.RTT.Milliseconds())/100.0)
	lossScore := 1.0 - m.LossRate
	bwScore := 0.5 // default moderate bandwidth score
	if m.Bandwidth > 0 {
		bwScore = float64(m.Bandwidth) / (float64(m.Bandwidth) + 1e6) // sigmoid around 1Mbps
	}

	return 0.4*rttScore + 0.35*lossScore + 0.25*bwScore
}
