package scheduler

import (
	"math"
	"sort"
)

// PathScore holds the computed score for a path.
type PathScore struct {
	PathID    string
	Type      PathType
	Score     float64
	Priority  int
	RTTMs     float64
	LossRate  float64
	Bandwidth float64
}

// priorityBonus returns the type-based priority bonus.
var priorityBonus = map[PathType]float64{
	PathUDPP2P:      1.0,
	PathQUICP2P:     0.8,
	PathTCPP2P:      0.6,
	PathNearbyRelay: 0.4,
	PathGlobalRelay: 0.2,
}

// ScorePath computes a composite quality score for a path.
// Higher score = better path.
func ScorePath(p *Path) PathScore {
	bonus := priorityBonus[p.Type]
	rttMs := float64(p.Metrics.RTT.Milliseconds())
	lossRate := p.Metrics.LossRate
	bwBps := float64(p.Metrics.Bandwidth)

	// RTT score: lower RTT = higher score
	rttScore := 1.0 / (1.0 + rttMs/100.0)

	// Loss score: lower loss = higher score
	lossScore := 1.0 - lossRate

	// Bandwidth score: higher BW = higher score (sigmoid around 1Mbps)
	bwScore := 0.5
	if bwBps > 0 {
		bwScore = bwBps / (bwBps + 1e6)
	}

	// Composite: weighted sum
	score := 0.30*bonus + 0.30*rttScore + 0.25*lossScore + 0.15*bwScore

	return PathScore{
		PathID:    p.ID,
		Type:      p.Type,
		Score:     math.Round(score*10000) / 10000,
		Priority:  p.Priority(),
		RTTMs:     rttMs,
		LossRate:  lossRate,
		Bandwidth: bwBps,
	}
}

// RankPaths sorts paths by score (highest first), breaking ties by priority.
func RankPaths(paths []*Path) []PathScore {
	scores := make([]PathScore, 0, len(paths))
	for _, p := range paths {
		if p.State != PathUnavailable {
			scores = append(scores, ScorePath(p))
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score != scores[j].Score {
			return scores[i].Score > scores[j].Score
		}
		return scores[i].Priority < scores[j].Priority // tie-break: lower priority number wins
	})

	return scores
}

// SelectBestPath returns the highest-scored path, or nil if none available.
func SelectBestPath(scores []PathScore) *PathScore {
	if len(scores) == 0 {
		return nil
	}
	return &scores[0]
}
