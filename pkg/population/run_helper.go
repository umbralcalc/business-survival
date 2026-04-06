package population

import (
	"github.com/umbralcalc/stochadex/pkg/simulator"
)

// RunToState executes a single-partition SingleLAPopulationIteration for steps
// monthly updates and returns the final state vector. init must match
// params-derived state width (nSectors * 60).
func RunToState(
	pMap map[string][]float64,
	init []float64,
	seed uint64,
	steps int,
) []float64 {
	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   init,
				Seed:              seed,
				StateWidth:        len(init),
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()
	store := simulator.NewStateTimeStorage()
	iter := &SingleLAPopulationIteration{}
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{iter},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: steps,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	for i := range impl.Iterations {
		impl.Iterations[i].Configure(i, settings)
	}
	coordinator := simulator.NewPartitionCoordinator(settings, impl)
	coordinator.Run()
	vals := store.GetValues("population")
	if len(vals) == 0 {
		return nil
	}
	out := make([]float64, len(init))
	copy(out, vals[len(vals)-1])
	return out
}
