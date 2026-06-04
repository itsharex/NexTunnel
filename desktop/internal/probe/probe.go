package probe

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Probe packet type markers.
const (
	probeTypeRequest byte = 0x01
	probeTypeReply   byte = 0x02
)

// Probeable is the interface a transport must satisfy for probing.
type Probeable interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// Prober performs periodic link quality probing over a transport.
type Prober struct {
	config    ProbeConfig
	transport Probeable
	metrics   atomic.Value // *LinkMetrics
	rttEst    *RTTEstimator
	lossTrack *LossTracker
	bwEst     *BandwidthEstimator

	observers []func(LinkMetrics)
	obsMu     sync.RWMutex

	seqNum uint16

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewProber creates a new link quality prober.
func NewProber(transport Probeable, opts ...ProbeOption) *Prober {
	cfg := DefaultProbeConfig()
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Prober{
		config:    cfg,
		transport: transport,
		rttEst:    NewRTTEstimator(cfg.RTTWindow),
		lossTrack: NewLossTracker(cfg.LossWindow),
		bwEst:     NewBandwidthEstimator(),
		ctx:       ctx,
		cancel:    cancel,
		logger:    cfg.Logger,
	}
	p.metrics.Store(&LinkMetrics{})
	return p
}

// Start begins the periodic probe loop.
func (p *Prober) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)
	go p.readEchoLoop()
	go p.probeLoop()
	p.logger.Info("prober started", "interval", p.config.Interval)
	return nil
}

// Stop stops the prober.
func (p *Prober) Stop() {
	p.cancel()
	p.logger.Info("prober stopped", "samples", p.rttEst.Count())
}

// Metrics returns the latest link metrics (atomic read).
func (p *Prober) Metrics() LinkMetrics {
	v := p.metrics.Load()
	if v == nil {
		return LinkMetrics{}
	}
	return *v.(*LinkMetrics)
}

// OnMetricsUpdate registers a callback for metric updates.
func (p *Prober) OnMetricsUpdate(fn func(LinkMetrics)) {
	p.obsMu.Lock()
	p.observers = append(p.observers, fn)
	p.obsMu.Unlock()
}

func (p *Prober) probeLoop() {
	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.sendProbe()
		}
	}
}

// sendProbe sends a single RTT probe.
// Format: [1B type | 2B seq | 1B flags | 8B timestamp_ns]
func (p *Prober) sendProbe() {
	p.seqNum++
	seq := p.seqNum

	pkt := make([]byte, 12)
	pkt[0] = probeTypeRequest
	binary.BigEndian.PutUint16(pkt[1:3], seq)
	pkt[3] = 0x00
	binary.BigEndian.PutUint64(pkt[4:12], uint64(time.Now().UnixNano()))

	p.lossTrack.RecordSent(seq)

	if _, err := p.transport.Write(pkt); err != nil {
		p.logger.Debug("probe send failed", "error", err)
	}
}

func (p *Prober) readEchoLoop() {
	buf := make([]byte, 1500)
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		n, err := p.transport.Read(buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			select {
			case <-p.ctx.Done():
				return
			default:
				continue
			}
		}

		if n < 12 {
			continue
		}

		switch buf[0] {
		case probeTypeRequest:
			// Echo back as reply
			reply := make([]byte, n)
			copy(reply, buf[:n])
			reply[0] = probeTypeReply
			p.transport.Write(reply)

		case probeTypeReply:
			seq := binary.BigEndian.Uint16(buf[1:3])
			sendNs := int64(binary.BigEndian.Uint64(buf[4:12]))
			sendTime := time.Unix(0, sendNs)
			rtt := time.Since(sendTime)

			// On Windows, time.Now() has ~15ms resolution.
			// If RTT is 0 (sub-tick), treat as < 1ms.
			if rtt <= 0 {
				rtt = 1 * time.Microsecond
			}

			p.lossTrack.RecordReceived(seq)
			p.rttEst.Update(rtt)

			if rtt > 0 {
				p.bwEst.AddSample(int64(n), rtt)
			}

			p.updateMetrics()
		}
	}
}

func (p *Prober) updateMetrics() {
	m := &LinkMetrics{
		PathID:      fmt.Sprintf("%v->%v", p.transport.LocalAddr(), p.transport.RemoteAddr()),
		RTT:         p.rttEst.SRTT(),
		RTTMin:      p.rttEst.Min(),
		RTTMax:      p.rttEst.Max(),
		RTTJitter:   p.rttEst.RTTVar(),
		LossRate:    p.lossTrack.LossRate(),
		Bandwidth:   p.bwEst.Estimate(),
		LastUpdated: time.Now(),
		SampleCount: p.rttEst.Count(),
	}

	p.metrics.Store(m)

	p.obsMu.RLock()
	observers := make([]func(LinkMetrics), len(p.observers))
	copy(observers, p.observers)
	p.obsMu.RUnlock()

	for _, fn := range observers {
		fn(*m)
	}
}
