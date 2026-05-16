package businessdash

import (
	"math"
	"testing"

	"github.com/umbralcalc/stochadex/pkg/simulator"
)

// TestSimulationRunsWithoutPanic exercises BuildBusinessSimulation
// through simulator.RunWithHarnesses — the project-convention sanity
// check for nil-pointer panics, NaN outputs, wrong state widths,
// params mutation, and state-history integrity.
func TestSimulationRunsWithoutPanic(t *testing.T) {
	gen := BuildBusinessSimulation()
	gen.SetSimulation(&simulator.SimulationConfig{
		OutputCondition: &simulator.EveryStepOutputCondition{},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 6,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: Stepsize},
		InitTimeValue:    0.0,
	})
	settings, implementations := gen.GenerateConfigs()
	simulator.RunWithHarnesses(settings, implementations)
}

// TestPopulationStaysPositive checks that the live stock count
// remains finite and non-negative across a short run — a NaN or
// negative would indicate the Leslie iteration's params got mangled.
func TestPopulationStaysPositive(t *testing.T) {
	gen := BuildBusinessSimulation()
	gen.SetSimulation(&simulator.SimulationConfig{
		OutputCondition: &simulator.EveryStepOutputCondition{},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 12,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: Stepsize},
		InitTimeValue:    0.0,
	})
	settings, implementations := gen.GenerateConfigs()

	store := simulator.NewStateTimeStorage()
	implementations.OutputFunction = &simulator.StateTimeStorageOutputFunction{Store: store}

	coordinator := simulator.NewPartitionCoordinator(settings, implementations)
	coordinator.Run()

	values := store.GetValues("stock_trajectory")
	if len(values) < 2 {
		t.Fatalf("expected at least 2 stock_trajectory outputs, got %d", len(values))
	}
	for i, row := range values {
		v := row[0]
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Errorf("step %d stock is non-finite: %v", i, v)
		}
		if v < 0 {
			t.Errorf("step %d stock is negative: %v", i, v)
		}
	}
}

// TestCohortDecaysFromOne checks that the cohort fraction starts at
// roughly 1.0 (the planted 5000-firm cohort) and decays monotonically
// in expectation — first decade values should be in [0, 1].
func TestCohortDecaysFromOne(t *testing.T) {
	gen := BuildBusinessSimulation()
	gen.SetSimulation(&simulator.SimulationConfig{
		OutputCondition: &simulator.EveryStepOutputCondition{},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 12,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: Stepsize},
		InitTimeValue:    0.0,
	})
	settings, implementations := gen.GenerateConfigs()

	store := simulator.NewStateTimeStorage()
	implementations.OutputFunction = &simulator.StateTimeStorageOutputFunction{Store: store}

	coordinator := simulator.NewPartitionCoordinator(settings, implementations)
	coordinator.Run()

	values := store.GetValues("cohort_trajectory")
	if len(values) < 2 {
		t.Fatalf("expected at least 2 cohort_trajectory outputs, got %d", len(values))
	}
	for i, row := range values {
		f := row[0]
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Errorf("step %d cohort fraction is non-finite: %v", i, f)
		}
		if f < 0 || f > 1.1 {
			t.Errorf("step %d cohort fraction out of [0, 1.1]: %v", i, f)
		}
	}
	// First post-init row should be slightly below 1.0 (a small
	// fraction of the cohort dies in month 0).
	first := values[1][0]
	if first < 0.9 || first > 1.0 {
		t.Errorf("cohort fraction after 1 month = %v; expected ~1.0", first)
	}
}

// TestPortfolioScatterCoversAllSixPoints checks that the
// portfolio_dots partition emits one (x, y, w, h) tuple per portfolio
// and that all six markers sit inside the scatter panel.
func TestPortfolioScatterCoversAllSixPoints(t *testing.T) {
	gen := BuildBusinessSimulation()
	gen.SetSimulation(&simulator.SimulationConfig{
		OutputCondition: &simulator.EveryStepOutputCondition{},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 3,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: Stepsize},
		InitTimeValue:    0.0,
	})
	settings, implementations := gen.GenerateConfigs()

	store := simulator.NewStateTimeStorage()
	implementations.OutputFunction = &simulator.StateTimeStorageOutputFunction{Store: store}

	coordinator := simulator.NewPartitionCoordinator(settings, implementations)
	coordinator.Run()

	rows := store.GetValues("portfolio_dots")
	if len(rows) == 0 {
		t.Fatal("expected portfolio_dots output rows, got none")
	}
	last := rows[len(rows)-1]
	if len(last) != NumPortfolios*4 {
		t.Fatalf("expected %d portfolio_dots values, got %d", NumPortfolios*4, len(last))
	}
	for i := 0; i < NumPortfolios; i++ {
		x := last[i*4+0]
		y := last[i*4+1]
		// Each marker's top-left corner should fall within the
		// scatter panel, allowing half a marker size of slack so the
		// marker can be centred on the axis edge.
		if x < float64(ScatterX)-float64(MarkerSize) || x > float64(ScatterX+ScatterWidth) {
			t.Errorf("portfolio %d marker x = %v outside [%d, %d]",
				i, x, ScatterX-MarkerSize, ScatterX+ScatterWidth)
		}
		if y < float64(ScatterY)-float64(MarkerSize) || y > float64(ScatterY+ScatterHeight) {
			t.Errorf("portfolio %d marker y = %v outside [%d, %d]",
				i, y, ScatterY-MarkerSize, ScatterY+ScatterHeight)
		}
	}
}
