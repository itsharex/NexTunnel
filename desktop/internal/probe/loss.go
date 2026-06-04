package probe

import "time"

// LossTracker computes packet loss rate over a sliding time window.
type LossTracker struct {
	window   time.Duration
	entries  []lossEntry
	sent     uint64
	received uint64
}

type lossEntry struct {
	seq      uint16
	sentAt   time.Time
	received bool
}

// NewLossTracker creates a loss tracker with the given sliding window.
func NewLossTracker(window time.Duration) *LossTracker {
	if window <= 0 {
		window = 30 * time.Second
	}
	return &LossTracker{
		window:  window,
		entries: make([]lossEntry, 0, 256),
	}
}

// RecordSent records that a probe packet was sent.
func (l *LossTracker) RecordSent(seq uint16) {
	l.sent++
	l.entries = append(l.entries, lossEntry{
		seq:    seq,
		sentAt: time.Now(),
	})
}

// RecordReceived records that a probe packet was received (acknowledged).
func (l *LossTracker) RecordReceived(seq uint16) {
	l.received++
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].seq == seq && !l.entries[i].received {
			l.entries[i].received = true
			break
		}
	}
}

// LossRate computes the packet loss rate over the sliding window.
// Returns a value between 0.0 (no loss) and 1.0 (complete loss).
func (l *LossTracker) LossRate() float64 {
	l.pruneOld()

	inWindow := 0
	lost := 0
	for _, e := range l.entries {
		inWindow++
		if !e.received {
			lost++
		}
	}

	if inWindow == 0 {
		return 0
	}
	return float64(lost) / float64(inWindow)
}

// TotalSent returns the total number of probe packets sent.
func (l *LossTracker) TotalSent() uint64 { return l.sent }

// TotalReceived returns the total number of probe packets received.
func (l *LossTracker) TotalReceived() uint64 { return l.received }

// pruneOld removes entries older than the sliding window.
func (l *LossTracker) pruneOld() {
	cutoff := time.Now().Add(-l.window)
	idx := 0
	for idx < len(l.entries) && l.entries[idx].sentAt.Before(cutoff) {
		idx++
	}
	if idx > 0 {
		l.entries = l.entries[idx:]
	}
}
