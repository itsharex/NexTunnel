package sdwan

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	// 1000 bytes/sec, burst = 2000
	tb := NewTokenBucket(1000)

	// Should allow first 2000 bytes (burst)
	if !tb.Allow(2000) {
		t.Fatal("expected burst allowance")
	}
	// Should reject immediately after burst
	if tb.Allow(100) {
		t.Fatal("expected rejection after burst exhausted")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(10000) // 10000 bytes/sec

	// Drain the bucket
	tb.Allow(20000)

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should have ~2000 tokens refilled (10000 * 0.2)
	avail := tb.Available()
	if avail < 1500 || avail > 3000 {
		t.Fatalf("expected ~2000 tokens after 200ms, got %.0f", avail)
	}
}

func TestTokenBucket_SetRate(t *testing.T) {
	tb := NewTokenBucket(1000)
	if tb.Rate() != 1000 {
		t.Fatalf("expected rate 1000, got %d", tb.Rate())
	}
	tb.SetRate(5000)
	if tb.Rate() != 5000 {
		t.Fatalf("expected rate 5000, got %d", tb.Rate())
	}
}

func TestTokenBucket_MinBurst(t *testing.T) {
	tb := NewTokenBucket(100) // very low rate
	// Burst should be at least 1500 (one MTU)
	if !tb.Allow(1500) {
		t.Fatal("expected minimum burst of 1500 bytes")
	}
}

func TestQoSManager_PriorityOrder(t *testing.T) {
	qm := NewQoSManager(100)

	// Enqueue packets in reverse priority order
	for i := QoSPriority(7); i >= 0; i-- {
		pkt := &Packet{FlowID: "flow-1", Size: 100, Priority: i}
		if !qm.Enqueue(pkt, 0) {
			t.Fatalf("failed to enqueue priority %d", i)
		}
	}

	// Dequeue should return in priority order (0 first)
	for i := QoSPriority(0); i < 8; i++ {
		pkt := qm.Dequeue()
		if pkt == nil {
			t.Fatalf("expected packet at priority %d", i)
		}
		if pkt.Priority != i {
			t.Fatalf("expected priority %d, got %d", i, pkt.Priority)
		}
	}

	// Queue should be empty
	if qm.Dequeue() != nil {
		t.Fatal("expected nil from empty queue")
	}
}

func TestQoSManager_RateLimiting(t *testing.T) {
	qm := NewQoSManager(100)

	// 500 bytes/sec limit
	limit := int64(500)

	// First packets should pass (burst = 1000)
	accepted := 0
	for i := 0; i < 20; i++ {
		pkt := &Packet{FlowID: "limited-flow", Size: 100, Priority: PriorityMedium}
		if qm.Enqueue(pkt, limit) {
			accepted++
		}
	}

	// Should accept ~10-15 (burst 1000 / 100 per packet + small refill during loop)
	if accepted < 8 || accepted > 16 {
		t.Fatalf("expected ~10-15 accepted packets, got %d", accepted)
	}
}

func TestQoSManager_QueueFull(t *testing.T) {
	qm := NewQoSManager(5) // very small queue

	for i := 0; i < 5; i++ {
		pkt := &Packet{FlowID: "flow", Size: 10, Priority: PriorityMedium}
		if !qm.Enqueue(pkt, 0) {
			t.Fatalf("should accept packet %d", i)
		}
	}

	// 6th packet should be rejected
	pkt := &Packet{FlowID: "flow", Size: 10, Priority: PriorityMedium}
	if qm.Enqueue(pkt, 0) {
		t.Fatal("expected rejection when queue is full")
	}

	stats := qm.GetStats()
	if stats.Dropped != 1 {
		t.Fatalf("expected 1 dropped, got %d", stats.Dropped)
	}
}

func TestQoSManager_DequeueBatch(t *testing.T) {
	qm := NewQoSManager(100)

	for i := 0; i < 10; i++ {
		pkt := &Packet{FlowID: "flow", Size: 50, Priority: QoSPriority(i % 8)}
		qm.Enqueue(pkt, 0)
	}

	batch := qm.DequeueBatch(5)
	if len(batch) != 5 {
		t.Fatalf("expected 5 packets, got %d", len(batch))
	}

	if qm.QueueLen() != 5 {
		t.Fatalf("expected 5 remaining, got %d", qm.QueueLen())
	}
}

func TestQoSManager_SetFlowLimit(t *testing.T) {
	qm := NewQoSManager(100)
	qm.SetFlowLimit("flow-a", 10000)
	qm.SetFlowLimit("flow-b", 20000)

	stats := qm.GetStats()
	if stats.FlowLimits != 2 {
		t.Fatalf("expected 2 flow limits, got %d", stats.FlowLimits)
	}

	qm.RemoveFlowLimit("flow-a")
	stats = qm.GetStats()
	if stats.FlowLimits != 1 {
		t.Fatalf("expected 1 flow limit, got %d", stats.FlowLimits)
	}
}

func TestQoSManager_Stats(t *testing.T) {
	qm := NewQoSManager(100)

	for i := 0; i < 5; i++ {
		pkt := &Packet{FlowID: "flow", Size: 100, Priority: PriorityHigh}
		qm.Enqueue(pkt, 0)
	}
	for i := 0; i < 3; i++ {
		qm.Dequeue()
	}

	stats := qm.GetStats()
	if stats.Enqueued != 5 {
		t.Fatalf("expected 5 enqueued, got %d", stats.Enqueued)
	}
	if stats.Dequeued != 3 {
		t.Fatalf("expected 3 dequeued, got %d", stats.Dequeued)
	}
	if stats.BytesIn != 500 {
		t.Fatalf("expected 500 bytes in, got %d", stats.BytesIn)
	}
	if stats.BytesOut != 300 {
		t.Fatalf("expected 300 bytes out, got %d", stats.BytesOut)
	}
}

func TestQoSManager_FIFOWithinPriority(t *testing.T) {
	qm := NewQoSManager(100)

	// Enqueue 3 packets with same priority but different flow IDs
	for _, id := range []string{"first", "second", "third"} {
		pkt := &Packet{FlowID: id, Size: 10, Priority: PriorityMedium}
		qm.Enqueue(pkt, 0)
		// Small delay to ensure different Enqueued times
		time.Sleep(1 * time.Millisecond)
	}

	// Should dequeue in FIFO order
	for _, expected := range []string{"first", "second", "third"} {
		pkt := qm.Dequeue()
		if pkt.FlowID != expected {
			t.Fatalf("expected %s, got %s", expected, pkt.FlowID)
		}
	}
}
