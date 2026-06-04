package probe

import (
	"math"
	"time"
)

// RTTEstimator computes smoothed RTT using EWMA with min-filter.
type RTTEstimator struct {
	alpha   float64       // smoothing factor (2/(N+1))
	srtt    time.Duration // smoothed RTT
	rttvar  time.Duration // RTT variance (jitter)
	rttMin  time.Duration // minimum observed RTT
	rttMax  time.Duration // maximum observed RTT
	count   int64
}

// NewRTTEstimator creates a new RTT estimator with the given window size.
func NewRTTEstimator(window int) *RTTEstimator {
	if window <= 0 {
		window = 32
	}
	alpha := 2.0 / float64(window+1)
	return &RTTEstimator{
		alpha:  alpha,
		rttMin: time.Duration(math.MaxInt64),
	}
}

// Update adds a new RTT sample and returns the updated smoothed RTT.
func (e *RTTEstimator) Update(rtt time.Duration) time.Duration {
	e.count++

	if rtt < e.rttMin {
		e.rttMin = rtt
	}
	if rtt > e.rttMax {
		e.rttMax = rtt
	}

	if e.count == 1 {
		// First sample: initialize directly
		e.srtt = rtt
		e.rttvar = rtt / 2
		return e.srtt
	}

	// EWMA: srtt = (1-alpha)*srtt + alpha*rtt
	e.srtt = time.Duration(float64(e.srtt)*(1-e.alpha) + float64(rtt)*e.alpha)

	// RTT variance: rttvar = (1-alpha)*rttvar + alpha*|rtt - srtt|
	diff := rtt - e.srtt
	if diff < 0 {
		diff = -diff
	}
	e.rttvar = time.Duration(float64(e.rttvar)*(1-e.alpha) + float64(diff)*e.alpha)

	return e.srtt
}

// SRTT returns the current smoothed RTT.
func (e *RTTEstimator) SRTT() time.Duration { return e.srtt }

// RTTVar returns the RTT variance (jitter).
func (e *RTTEstimator) RTTVar() time.Duration { return e.rttvar }

// Min returns the minimum observed RTT.
func (e *RTTEstimator) Min() time.Duration {
	if e.count == 0 {
		return 0
	}
	return e.rttMin
}

// Max returns the maximum observed RTT.
func (e *RTTEstimator) Max() time.Duration { return e.rttMax }

// Count returns the number of samples.
func (e *RTTEstimator) Count() int64 { return e.count }
