package sdwan

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

// TokenBucket implements a token bucket rate limiter for bandwidth control.
// Tokens represent bytes; the bucket refills at a configurable rate (bytes/sec).
type TokenBucket struct {
	mu         sync.Mutex
	rate       int64   // bytes per second (refill rate)
	burst      int64   // maximum bucket capacity
	tokens     float64 // current available tokens
	lastRefill time.Time
}

// NewTokenBucket creates a token bucket with the given rate (bytes/sec).
// Burst defaults to 2× the rate to allow short traffic spikes.
func NewTokenBucket(rateBytesPerSec int64) *TokenBucket {
	burst := rateBytesPerSec * 2
	if burst < 1500 {
		burst = 1500 // at least one MTU
	}
	return &TokenBucket{
		rate:       rateBytesPerSec,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

// Allow checks whether n bytes can be sent. If yes, consumes the tokens and returns true.
// If no, returns false without consuming tokens.
func (tb *TokenBucket) Allow(n int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// Rate returns the current rate limit in bytes/sec.
func (tb *TokenBucket) Rate() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.rate
}

// SetRate updates the rate limit dynamically without resetting the bucket.
func (tb *TokenBucket) SetRate(rateBytesPerSec int64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.rate = rateBytesPerSec
	tb.burst = rateBytesPerSec * 2
	if tb.burst < 1500 {
		tb.burst = 1500
	}
}

// Available returns the current number of available tokens (bytes).
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	tb.tokens += elapsed * float64(tb.rate)
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
	tb.lastRefill = now
}

// Packet represents a network packet enqueued for QoS scheduling.
type Packet struct {
	FlowID   string
	NodeID   string
	Size     int
	Priority QoSPriority
	Enqueued time.Time
	Data     []byte // optional payload reference
}

// priorityQueue implements heap.Interface for Packet pointers.
type priorityQueue []*Packet

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Lower priority number = higher priority (processed first)
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority < pq[j].Priority
	}
	// Same priority: FIFO by enqueue time
	return pq[i].Enqueued.Before(pq[j].Enqueued)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Packet))
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[:n-1]
	return item
}

// QoSManager manages per-flow rate limiters and a global priority queue.
// It supports 8 priority levels (0=Critical … 7=Background).
type QoSManager struct {
	mu      sync.Mutex
	queue   priorityQueue
	limiters map[string]*TokenBucket // flowID -> limiter

	// stats
	enqueued   atomic.Uint64
	dequeued   atomic.Uint64
	dropped    atomic.Uint64
	bytesIn    atomic.Uint64
	bytesOut   atomic.Uint64

	maxQueueSize int
}

// NewQoSManager creates a QoS manager with the given max queue capacity.
func NewQoSManager(maxQueueSize int) *QoSManager {
	qm := &QoSManager{
		queue:        make(priorityQueue, 0),
		limiters:     make(map[string]*TokenBucket),
		maxQueueSize: maxQueueSize,
	}
	heap.Init(&qm.queue)
	return qm
}

// Enqueue adds a packet to the priority queue, applying rate limiting.
// Returns false if the queue is full or the flow exceeds its bandwidth limit.
func (qm *QoSManager) Enqueue(pkt *Packet, bandwidthLimit int64) bool {
	// Check rate limit before enqueueing
	if bandwidthLimit > 0 {
		limiter := qm.getOrCreateLimiter(pkt.FlowID, bandwidthLimit)
		if !limiter.Allow(int64(pkt.Size)) {
			qm.dropped.Add(1)
			return false
		}
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	if qm.queue.Len() >= qm.maxQueueSize {
		qm.dropped.Add(1)
		return false
	}

	pkt.Enqueued = time.Now()
	heap.Push(&qm.queue, pkt)
	qm.enqueued.Add(1)
	qm.bytesIn.Add(uint64(pkt.Size))
	return true
}

// Dequeue removes and returns the highest-priority packet from the queue.
// Returns nil if the queue is empty.
func (qm *QoSManager) Dequeue() *Packet {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if qm.queue.Len() == 0 {
		return nil
	}

	pkt := heap.Pop(&qm.queue).(*Packet)
	qm.dequeued.Add(1)
	qm.bytesOut.Add(uint64(pkt.Size))
	return pkt
}

// DequeueBatch removes up to count packets from the queue.
func (qm *QoSManager) DequeueBatch(count int) []*Packet {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	n := count
	if n > qm.queue.Len() {
		n = qm.queue.Len()
	}
	if n == 0 {
		return nil
	}

	result := make([]*Packet, n)
	for i := 0; i < n; i++ {
		pkt := heap.Pop(&qm.queue).(*Packet)
		qm.dequeued.Add(1)
		qm.bytesOut.Add(uint64(pkt.Size))
		result[i] = pkt
	}
	return result
}

// QueueLen returns the current number of packets in the queue.
func (qm *QoSManager) QueueLen() int {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	return qm.queue.Len()
}

// SetFlowLimit updates the bandwidth limit for a specific flow.
func (qm *QoSManager) SetFlowLimit(flowID string, rateBytesPerSec int64) {
	qm.mu.Lock()
	limiter, ok := qm.limiters[flowID]
	qm.mu.Unlock()

	if ok {
		limiter.SetRate(rateBytesPerSec)
	} else {
		qm.getOrCreateLimiter(flowID, rateBytesPerSec)
	}
}

// RemoveFlowLimit removes the rate limiter for a flow.
func (qm *QoSManager) RemoveFlowLimit(flowID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	delete(qm.limiters, flowID)
}

// Stats returns QoS manager statistics.
type QoSStats struct {
	QueueLen   int    `json:"queue_len"`
	Enqueued   uint64 `json:"enqueued"`
	Dequeued   uint64 `json:"dequeued"`
	Dropped    uint64 `json:"dropped"`
	BytesIn    uint64 `json:"bytes_in"`
	BytesOut   uint64 `json:"bytes_out"`
	FlowLimits int    `json:"flow_limits"`
}

// GetStats returns current QoS statistics.
func (qm *QoSManager) GetStats() QoSStats {
	qm.mu.Lock()
	qLen := qm.queue.Len()
	nLimiters := len(qm.limiters)
	qm.mu.Unlock()

	return QoSStats{
		QueueLen:   qLen,
		Enqueued:   qm.enqueued.Load(),
		Dequeued:   qm.dequeued.Load(),
		Dropped:    qm.dropped.Load(),
		BytesIn:    qm.bytesIn.Load(),
		BytesOut:   qm.bytesOut.Load(),
		FlowLimits: nLimiters,
	}
}

func (qm *QoSManager) getOrCreateLimiter(flowID string, rate int64) *TokenBucket {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if limiter, ok := qm.limiters[flowID]; ok {
		return limiter
	}
	limiter := NewTokenBucket(rate)
	qm.limiters[flowID] = limiter
	return limiter
}
