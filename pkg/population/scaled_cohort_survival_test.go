package population

import (
	"testing"

	"github.com/umbralcalc/stochadex/pkg/simulator"
	"gonum.org/v1/gonum/floats/scalar"
)

func TestScaledCohortSurvivalIteration_UnityScale(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name: "fwd",
				Params: simulator.NewParams(map[string][]float64{
					"survival_fracs": surv,
					"param_values":   {1.0},
				}),
				InitStateValues:   []float64{0},
				Seed:              1,
				StateWidth:        1,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()
	iter := &ScaledCohortSurvivalIteration{}
	iter.Configure(0, settings)
	out := iter.Iterate(&settings.Iterations[0].Params, 0, nil, nil)
	want := CumulativeSurvivalAfterMonths(MonthlyHazardsFromCumulativeSurvival(surv), 60)
	if !scalar.EqualWithinAbs(out[0], want, 1e-9) {
		t.Fatalf("got %f want %f", out[0], want)
	}
}

func TestScaledCohortSurvivalIteration_RunWithHarnesses(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name: "scaled",
				Params: simulator.NewParams(map[string][]float64{
					"survival_fracs": surv,
					"param_values":   {1.0},
				}),
				InitStateValues:   []float64{0},
				Seed:              42,
				StateWidth:        1,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{&ScaledCohortSurvivalIteration{}},
		OutputCondition: &simulator.NilOutputCondition{},
		OutputFunction:  &simulator.NilOutputFunction{},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 8,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	if err := simulator.RunWithHarnesses(settings, impl); err != nil {
		t.Fatal(err)
	}
}
