package population

import (
	"math"

	"github.com/umbralcalc/stochadex/pkg/simulator"
)

// PopulationSurvivalBirthMomentsIteration is a one-step forward statistic for SMC:
// it reads param_values[0] (hazard multiplier) and param_values[1] (birth
// multiplier), runs a deterministic single-sector monthly Leslie for 60 months
// with an isolated initial cohort of mass 1, and returns:
//
//	[ five_year_cohort_survival, mean_monthly_births ]
//
// Configure requires state_width == 2, survival_fracs (length 5), and
// base_birth_rate_scalar (mean monthly births at multiplier 1).
type PopulationSurvivalBirthMomentsIteration struct {
	survivalFracs []float64
	baseBirth     float64
	width         int
}

func (p *PopulationSurvivalBirthMomentsIteration) Configure(partitionIndex int, settings *simulator.Settings) {
	is := settings.Iterations[partitionIndex]
	p.width = is.StateWidth
	if p.width != 2 {
		panic("population: PopulationSurvivalBirthMomentsIteration state_width must be 2")
	}
	p.survivalFracs = append([]float64(nil), is.Params.Get("survival_fracs")...)
	if len(p.survivalFracs) != 5 {
		panic("population: need 5 survival_fracs")
	}
	p.baseBirth = is.Params.GetIndex("base_birth_rate_scalar", 0)
}

func (p *PopulationSurvivalBirthMomentsIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	hazardScale := params.GetIndex("param_values", 0)
	if hazardScale < 0 {
		hazardScale = 0
	}
	birthScale := params.GetIndex("param_values", 1)
	if birthScale < 0 {
		birthScale = 0
	}
	baseH := MonthlyHazardsFromCumulativeSurvival(p.survivalFracs)

	// Isolated cohort (no entrants) — mirrors deterministic Leslie aging.
	c := make([]float64, 60)
	c[0] = 1.0
	var birthAccum float64
	for step := 0; step < 60; step++ {
		birthAccum += p.baseBirth * birthScale
		next := make([]float64, 60)
		// age 1..58
		for age := 1; age <= 58; age++ {
			h := baseH[age-1] * hazardScale
			if h < 0 {
				h = 0
			}
			if h > 1 {
				h = 1
			}
			next[age] += c[age-1] * (1.0 - h)
		}
		h58 := baseH[58] * hazardScale
		if h58 < 0 {
			h58 = 0
		}
		if h58 > 1 {
			h58 = 1
		}
		h59 := baseH[59] * hazardScale
		if h59 < 0 {
			h59 = 0
		}
		if h59 > 1 {
			h59 = 1
		}
		next[59] += c[58]*(1.0-h58) + c[59]*(1.0-h59)
		c = next
	}
	var surv float64
	for _, v := range c {
		surv += v
	}
	if math.IsNaN(surv) {
		surv = 0
	}
	meanBirths := birthAccum / 60.0
	if math.IsNaN(meanBirths) {
		meanBirths = 0
	}
	return []float64{surv, meanBirths}
}
