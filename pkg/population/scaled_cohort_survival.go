package population

import (
	"math"

	"github.com/umbralcalc/stochadex/pkg/simulator"
)

// ScaledCohortSurvivalIteration maps param_values[0] (a non-negative global hazard
// multiplier) to deterministic 60-month cumulative survival under the baseline
// ONS-style cumulative survival fractions in params "survival_fracs" (length 5).
//
// Used as the "forward model" inside SMC / embedded simulations: one step
// outputs a single statistic for Gaussian likelihood comparison to a data target.
type ScaledCohortSurvivalIteration struct {
	survivalFracs []float64
}

func (s *ScaledCohortSurvivalIteration) Configure(partitionIndex int, settings *simulator.Settings) {
	s.survivalFracs = append([]float64(nil),
		settings.Iterations[partitionIndex].Params.Get("survival_fracs")...)
}

func (s *ScaledCohortSurvivalIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	scale := params.GetIndex("param_values", 0)
	if scale < 0 {
		scale = 0
	}
	base := MonthlyHazardsFromCumulativeSurvival(s.survivalFracs)
	h := make([]float64, len(base))
	for i := range base {
		h[i] = base[i] * scale
		if h[i] < 0 {
			h[i] = 0
		}
		if h[i] > 1 {
			h[i] = 1
		}
	}
	surv := CumulativeSurvivalAfterMonths(h, 60)
	if math.IsNaN(surv) {
		surv = 0
	}
	return []float64{surv}
}
