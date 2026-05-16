package evaluate

import "math"

// DistressHazardBoostFromClaimants builds a nonnegative series aligned with
// claimant count months: rolling volatility of log-differences scaled into
// small hazard boosts (Companies-House-style distress proxy without filings).
func DistressHazardBoostFromClaimants(claimants []float64, window int) []float64 {
	if window < 3 {
		window = 6
	}
	n := len(claimants)
	out := make([]float64, n)
	if n == 0 {
		return out
	}
	dl := make([]float64, n)
	for i := 1; i < n; i++ {
		a := math.Log(math.Max(1, claimants[i]))
		b := math.Log(math.Max(1, claimants[i-1]))
		dl[i] = a - b
	}
	for i := range out {
		lo := i - window + 1
		if lo < 0 {
			lo = 0
		}
		var sum, sumsq float64
		k := 0
		for j := lo; j <= i; j++ {
			x := dl[j]
			sum += x
			sumsq += x * x
			k++
		}
		if k < 2 {
			continue
		}
		mean := sum / float64(k)
		variance := sumsq/float64(k) - mean*mean
		if variance < 0 {
			variance = 0
		}
		sd := math.Sqrt(variance)
		out[i] = math.Min(0.35, 8.0*sd)
	}
	return out
}
