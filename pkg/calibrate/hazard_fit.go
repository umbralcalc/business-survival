package calibrate

import (
	"github.com/umbralcalc/business-survival/pkg/population"
)

// DefaultSectorHazardRelatives are literature-style priors: hazard multipliers
// vs a baseline sector (Professional = 1) before global calibration.
var DefaultSectorHazardRelatives = map[string]float64{
	"Professional":  1.0,
	"Technology":    0.92,
	"Construction":  1.05,
	"Retail":        1.1,
	"Hospitality":   1.28,
	"Other":         1.0,
}

// BlendMonthlyHazard mixes per-sector scaled baseline hazards using live-register
// sector shares. mix keys must match sector labels used in explore/lifecycle.
func BlendMonthlyHazard(
	baseMonthly []float64,
	mix map[string]float64,
	relatives map[string]float64,
	globalScale float64,
) []float64 {
	out := make([]float64, len(baseMonthly))
	for m := range baseMonthly {
		var w float64
		for sec, share := range mix {
			r := relatives[sec]
			if r == 0 {
				r = 1.0
			}
			w += share * r
		}
		h := globalScale * w * baseMonthly[m]
		if h < 0 {
			h = 0
		}
		if h > 1 {
			h = 1
		}
		out[m] = h
	}
	return out
}

// FitGlobalHazardScale searches a scalar globalScale such that the 60-month
// cumulative survival under the blended hazard matches target5yr (e.g. ONS
// UK five-year survival fraction). Requires survivorsFrac in (0,1).
func FitGlobalHazardScale(
	survivalFracsYears []float64,
	mix map[string]float64,
	relatives map[string]float64,
	target5yr float64,
) float64 {
	base := population.MonthlyHazardsFromCumulativeSurvival(survivalFracsYears)
	if len(base) != 60 {
		return 1.0
	}
	lo, hi := 1e-8, 50.0
	for iter := 0; iter < 48; iter++ {
		mid := (lo + hi) / 2
		hz := BlendMonthlyHazard(base, mix, relatives, mid)
		s := population.CumulativeSurvivalAfterMonths(hz, 60)
		if s < target5yr {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2
}
