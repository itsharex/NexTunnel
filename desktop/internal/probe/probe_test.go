package probe

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// pipeTransport uses channels for data flow.
type pipeTransport struct {
	readCh  chan []byte
	writeCh chan []byte
	local   net.Addr
	remote  net.Addr
	closeCh chan struct{}
}

func newPipePair() (Probeable, Probeable) {
	ab := make(chan []byte, 256)
	ba := make(chan []byte, 256)
	a := &pipeTransport{
		readCh: ba, writeCh: ab,
		local:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001},
		remote:  &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002},
		closeCh: make(chan struct{}),
	}
	b := &pipeTransport{
		readCh: ab, writeCh: ba,
		local:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002},
		remote:  &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001},
		closeCh: make(chan struct{}),
	}
	return a, b
}

func (p *pipeTransport) Read(buf []byte) (int, error) {
	select {
	case data, ok := <-p.readCh:
		if !ok {
			return 0, context.Canceled
		}
		return copy(buf, data), nil
	case <-p.closeCh:
		return 0, context.Canceled
	case <-time.After(5 * time.Second):
		return 0, context.DeadlineExceeded
	}
}

func (p *pipeTransport) Write(data []byte) (int, error) {
	d := make([]byte, len(data))
	copy(d, data)
	select {
	case p.writeCh <- d:
		return len(data), nil
	case <-p.closeCh:
		return 0, context.Canceled
	}
}

func (p *pipeTransport) Close() error {
	select {
	case <-p.closeCh:
	default:
		close(p.closeCh)
	}
	return nil
}

func (p *pipeTransport) LocalAddr() net.Addr  { return p.local }
func (p *pipeTransport) RemoteAddr() net.Addr { return p.remote }

// --- Unit tests ---

func TestRTTEstimator_EWMA(t *testing.T) {
	est := NewRTTEstimator(8)
	est.Update(10 * time.Millisecond)
	if est.SRTT() != 10*time.Millisecond {
		t.Errorf("SRTT = %v, want 10ms", est.SRTT())
	}
	est.Update(20 * time.Millisecond)
	if srtt := est.SRTT(); srtt <= 10*time.Millisecond || srtt >= 20*time.Millisecond {
		t.Errorf("SRTT = %v, want 10-20ms", srtt)
	}
	for i := 0; i < 10; i++ {
		est.Update(15 * time.Millisecond)
	}
	if est.Count() != 12 {
		t.Errorf("Count = %d, want 12", est.Count())
	}
	t.Logf("SRTT=%v Min=%v Max=%v", est.SRTT(), est.Min(), est.Max())
}

func TestLossTracker_SlidingWindow(t *testing.T) {
	tracker := NewLossTracker(10 * time.Second)
	for i := uint16(1); i <= 10; i++ {
		tracker.RecordSent(i)
		if i <= 8 {
			tracker.RecordReceived(i)
		}
	}
	if rate := tracker.LossRate(); rate < 0.19 || rate > 0.21 {
		t.Errorf("LossRate = %f, want ~0.2", rate)
	}
	if tracker.TotalSent() != 10 || tracker.TotalReceived() != 8 {
		t.Errorf("sent=%d recv=%d", tracker.TotalSent(), tracker.TotalReceived())
	}
}

func TestBandwidthEstimator(t *testing.T) {
	bw := NewBandwidthEstimator()
	bw.AddSample(65536, 1*time.Millisecond)
	bw.AddSample(65536, 2*time.Millisecond)
	if est := bw.Estimate(); est <= 0 {
		t.Errorf("bandwidth = %d, want > 0", est)
	}
	t.Logf("BW: %d bps", bw.Estimate())
}

func TestProber_Integration(t *testing.T) {
	a, b := newPipePair()
	defer a.Close()
	defer b.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proberA := NewProber(a, WithInterval(50*time.Millisecond))
	proberA.Start(ctx)

	// Echo handler on B side
	go func() {
		buf := make([]byte, 1500)
		for {
			n, err := b.Read(buf)
			if err != nil {
				return
			}
			if n >= 12 && buf[0] == probeTypeRequest {
				reply := make([]byte, n)
				copy(reply, buf[:n])
				reply[0] = probeTypeReply
				b.Write(reply)
			}
		}
	}()

	time.Sleep(400 * time.Millisecond)
	cancel()
	proberA.Stop()

	m := proberA.Metrics()
	t.Logf("Metrics: RTT=%v Loss=%.2f Samples=%d", m.RTT, m.LossRate, m.SampleCount)

	if m.SampleCount == 0 {
		t.Fatal("no samples collected")
	}
	if m.RTT <= 0 {
		t.Error("RTT should be > 0 (>= 1us for sub-tick)")
	}
	if m.LossRate > 0.5 {
		t.Errorf("LossRate = %f, expected low", m.LossRate)
	}
	if !m.IsReachable() {
		t.Error("path should be reachable")
	}
}

func TestProbePacketRoundtrip(t *testing.T) {
	// Test with an explicit time gap to verify encoding
	sendTime := time.Now().Add(-5 * time.Millisecond) // simulate 5ms ago
	pkt := make([]byte, 12)
	pkt[0] = probeTypeRequest
	binary.BigEndian.PutUint16(pkt[1:3], 42)
	binary.BigEndian.PutUint64(pkt[4:12], uint64(sendTime.UnixNano()))

	// Decode
	seq := binary.BigEndian.Uint16(pkt[1:3])
	sendNs := int64(binary.BigEndian.Uint64(pkt[4:12]))
	decodedTime := time.Unix(0, sendNs)
	rtt := time.Since(decodedTime)

	if seq != 42 {
		t.Fatalf("seq = %d, want 42", seq)
	}
	if rtt < 4*time.Millisecond {
		t.Errorf("RTT = %v, want >= 4ms", rtt)
	}
	t.Logf("Packet roundtrip: seq=%d RTT=%v", seq, rtt)

	est := NewRTTEstimator(8)
	est.Update(rtt)
	if est.SRTT() <= 0 {
		t.Error("SRTT should be > 0")
	}
	t.Logf("SRTT: %v", est.SRTT())
}

func TestLinkMetrics_QualityScore(t *testing.T) {
	good := &LinkMetrics{RTT: 10 * time.Millisecond, LossRate: 0.0, Bandwidth: 10e6}
	bad := &LinkMetrics{RTT: 200 * time.Millisecond, LossRate: 0.1, Bandwidth: 1e6}
	dead := &LinkMetrics{RTT: 0}
	sg := good.QualityScore()
	sb := bad.QualityScore()
	if sg <= 0 || sg > 1 {
		t.Errorf("good score = %f", sg)
	}
	if sb >= sg {
		t.Errorf("bad %f >= good %f", sb, sg)
	}
	if dead.IsReachable() {
		t.Error("zero RTT not reachable")
	}
	t.Logf("Good=%.3f Bad=%.3f", sg, sb)
}
