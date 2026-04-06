package calibrate

import "math"

// SimulationElasticitiesFromPanel maps pooled first-difference regression
// coefficients (Δ births ~ β_r Δr + β_c Δ log claimants) onto the
// exponential elasticities used by population.SingleLAPopulationIteration:
//
//	birthMult ≈ exp(eRate*(r-rRef) + eClaim*log(c/cRef))
//
// We use a local linearisation around pooled means: β_r ≈ eRate * meanBirthLevel
// on comparable monthly scales is a rough guide only — scale down aggressively
// for stability in simulation.
//
// deathRateElasticity is a heuristic default when recessions lift exits via
// financing stress (not from the same FD regression).
func SimulationElasticitiesFromPanel(panelPath string, meanMonthlyBirthsLA float64) (eRate, eClaim, deathRateElasticity float64, err error) {
	bR, bC, n, err := PooledFirstDiffRegression(panelPath)
	if err != nil {
		return 0, 0, 0, err
	}
	if n < 10 {
		return 0, 0, 0.12, nil
	}
	mb := meanMonthlyBirthsLA
	if mb < 1 {
		mb = 500
	}
	// Scale FD beta (delta births per 1 unit rate change) to ~semielasticity.
	den := math.Max(mb, 20.0)
	eRate = bR / den
	eClaim = bC / den
	// Cap to avoid explosive birthMult in short panels with volatile FD.
	if eRate > 0 {
		eRate = math.Min(eRate, 0.15)
	} else {
		eRate = math.Max(eRate, -0.35)
	}
	if eClaim > 0 {
		eClaim = math.Min(eClaim, 0.25)
	} else {
		eClaim = math.Max(eClaim, -0.4)
	}

	_, meanCovid, meanRest, _, _, _, err := PanelRecessionWindows(panelPath)
	deathRateElasticity = 0.12
	if err == nil && meanCovid < meanRest-0.5 {
		deathRateElasticity = 0.18
	}
	return eRate, eClaim, deathRateElasticity, nil
}
