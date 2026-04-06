package population

import (
	"testing"

	"github.com/umbralcalc/stochadex/pkg/simulator"
	"gonum.org/v1/gonum/floats/scalar"
)

func TestPopulationMoments_MatchesScaledCohortOnSurvival(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	scale := 1.05
	pMap := map[string][]float64{
		"survival_fracs":            surv,
		"param_values":              {scale},
	}
	set := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   []float64{0},
				StateWidth:        1,
				StateHistoryDepth: 2,
				Seed:              1,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	set.Init()
	sc := &ScaledCohortSurvivalIteration{}
	sc.Configure(0, set)
	outCohort := sc.Iterate(&simulator.Params{Map: pMap}, 0, nil, nil)

	mMap := map[string][]float64{
		"survival_fracs":          surv,
		"base_birth_rate_scalar":  {0},
		"param_values":            {scale, 0},
	}
	mset := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Params:            simulator.Params{Map: mMap},
				InitStateValues:   []float64{0, 0},
				StateWidth:        2,
				StateHistoryDepth: 2,
				Seed:              2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	mset.Init()
	mom := &PopulationSurvivalBirthMomentsIteration{}
	mom.Configure(0, mset)
	outMom := mom.Iterate(&simulator.Params{Map: mMap}, 0, nil, nil)

	if !scalar.EqualWithinAbs(outMom[0], outCohort[0], 1e-9) {
		t.Fatalf("survival: moments=%v scaled=%v", outMom[0], outCohort[0])
	}
	if !scalar.EqualWithinAbs(outMom[1], 0, 1e-9) {
		t.Fatalf("mean births: %v", outMom[1])
	}
}

func TestPopulationMoments_BirthScaleLinear(t *testing.T) {
	surv := []float64{0.95, 0.75, 0.55, 0.44, 0.38}
	mMap := map[string][]float64{
		"survival_fracs":         surv,
		"base_birth_rate_scalar": {10},
		"param_values":           {1.0, 2.0},
	}
	mset := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Params:            simulator.Params{Map: mMap},
				InitStateValues:   []float64{0, 0},
				StateWidth:        2,
				StateHistoryDepth: 2,
				Seed:              3,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	mset.Init()
	mom := &PopulationSurvivalBirthMomentsIteration{}
	mom.Configure(0, mset)
	out := mom.Iterate(&simulator.Params{Map: mMap}, 0, nil, nil)
	if !scalar.EqualWithinAbs(out[1], 20.0, 1e-9) {
		t.Fatalf("mean monthly births: got %v want 20", out[1])
	}
}
