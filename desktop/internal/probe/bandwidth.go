package probe

import "time"

// BandwidthEstimator estimates link bandwidth using packet-pair dispersion.
// It sends back-to-back packet pairs and measures the arrival time gap
// at the receiver to estimate bottleneck bandwidth.
type BandwidthEstimator struct {
	samples  []int64 // bandwidth samples in bps
	maxStore int
}

// NewBandwidthEstimator creates a new bandwidth estimator.
func NewBandwidthEstimator() *BandwidthEstimator {
	return &BandwidthEstimator{
		samples:  make([]int64, 0, 64),
		maxStore: 64,
	}
}

// AddSample adds a bandwidth sample computed from a packet pair.
// packetSize is the size of one packet in bytes.
// dispersion is the time gap between receiving two consecutive packets.
func (b *BandwidthEstimator) AddSample(packetSize int64, dispersion time.Duration) {
	if dispersion <= 0 {
		return
	}

	// bandwidth = packetSize * 8 bits / dispersion_seconds
	bps := int64(float64(packetSize*8) / dispersion.Seconds())
	if bps <= 0 {
		return
	}

	b.samples = append(b.samples, bps)
	if len(b.samples) > b.maxStore {
		b.samples = b.samples[1:]
	}
}

// Estimate returns the median bandwidth estimate in bits per second.
func (b *BandwidthEstimator) Estimate() int64 {
	n := len(b.samples)
	if n == 0 {
		return 0
	}

	// Sort a copy for median calculation
	sorted := make([]int64, n)
	copy(sorted, b.samples)
	sortInt64s(sorted)

	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// Latest returns the most recent bandwidth sample, or 0.
func (b *BandwidthEstimator) Latest() int64 {
	if len(b.samples) == 0 {
		return 0
	}
	return b.samples[len(b.samples)-1]
}

// Count returns the number of samples.
func (b *BandwidthEstimator) Count() int { return len(b.samples) }

// sortInt64s sorts a slice of int64 in ascending order (insertion sort for small slices).
func sortInt64s(a []int64) {
	for i := 1; i < len(a); i++ {
		key := a[i]
		j := i - 1
		for j >= 0 && a[j] > key {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = key
	}
}
