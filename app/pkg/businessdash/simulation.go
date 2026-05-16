package businessdash

import (
	"github.com/umbralcalc/business-survival/pkg/policy"
	"github.com/umbralcalc/stochadex/pkg/simulator"
)

// BuildBusinessSimulation constructs the stochadex generator for the
// business-survival dashboard.
//
// The partition graph wires:
//
//	policy_action          slider/radio input ([portfolio_idx, scenario_idx])
//	population             live Leslie population — 6 sectors × 60 age buckets
//	cohort                 5000-business cohort planted at age 0, births off
//	stock_trajectory       single-scalar stock count for the live line chart
//	cohort_trajectory      single-scalar cohort surviving fraction for the line chart
//	portfolio_dots         scatter markers for all six portfolios under the
//	                       currently-selected scenario
//	portfolio_highlight    larger marker at the user's selected portfolio
//	display_progress       readout: month count + live stock
//	display_outcomes       readout: README reference survival % + final stock
//
// The population and cohort partitions share the same LesliePopulationIteration
// machinery — the cohort instance turns births off so the planted cohort
// just decays through the age buckets.
func BuildBusinessSimulation() *simulator.ConfigGenerator {
	loadHullParams()
	ensurePortfolioTable()

	nSec := len(policy.SectorOrder)
	width := nSec * 60

	// Seed initial population state: spread the per-sector total across
	// all 60 age buckets evenly. This gives the live trajectory a
	// realistic starting stock instead of cold-starting from zero (the
	// Leslie iteration would otherwise take a year+ to fill the ageing
	// pyramid).
	initStock := make([]float64, width)
	for sec := 0; sec < nSec; sec++ {
		perAge := hullParams.InitialStockPerSector[sec] / 60.0
		for age := 0; age < 60; age++ {
			initStock[sec*60+age] = perAge
		}
	}

	// Seed initial cohort: plant CohortSize businesses at age 0,
	// distributed across sectors in proportion to the base birth rate
	// mix. This matches the project's offline cohort sub-run.
	initCohort := make([]float64, width)
	totalBirth := 0.0
	for _, b := range hullParams.BaseBirthRates {
		totalBirth += b
	}
	for sec := 0; sec < nSec; sec++ {
		share := 0.0
		if totalBirth > 0 {
			share = hullParams.BaseBirthRates[sec] / totalBirth
		}
		initCohort[sec*60+0] = hullParams.CohortSize * share
	}

	policyAction := &simulator.PartitionConfig{
		Name:      "policy_action",
		Iteration: &PolicyActionIteration{},
		Params: simulator.NewParams(map[string][]float64{
			"action_state_values": {
				PortfolioBaseline,
				ScenarioBaseline,
			},
		}),
		InitStateValues: []float64{
			PortfolioBaseline,
			ScenarioBaseline,
		},
		StateHistoryDepth: 1,
		Seed:              0,
	}

	population := &simulator.PartitionConfig{
		Name:      "population",
		Iteration: NewLesliePopulationIteration(false),
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"policy_action": {Upstream: "policy_action"},
		},
		InitStateValues:   initStock,
		StateHistoryDepth: 2,
		Seed:              17,
	}

	cohort := &simulator.PartitionConfig{
		Name:      "cohort",
		Iteration: NewLesliePopulationIteration(true),
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"policy_action": {Upstream: "policy_action"},
		},
		InitStateValues:   initCohort,
		StateHistoryDepth: 2,
		Seed:              23,
	}

	stockTrajectory := &simulator.PartitionConfig{
		Name:      "stock_trajectory",
		Iteration: &StockTrajectoryIteration{},
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"population_state": {Upstream: "population"},
		},
		InitStateValues:   []float64{0},
		StateHistoryDepth: 1,
		Seed:              0,
	}

	cohortTrajectory := &simulator.PartitionConfig{
		Name:      "cohort_trajectory",
		Iteration: &CohortTrajectoryIteration{},
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"cohort_state": {Upstream: "cohort"},
		},
		InitStateValues:   []float64{1.0},
		StateHistoryDepth: 1,
		Seed:              0,
	}

	portfolioDots := &simulator.PartitionConfig{
		Name:      "portfolio_dots",
		Iteration: &PortfolioDotsIteration{},
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"policy_action": {Upstream: "policy_action"},
		},
		InitStateValues:   make([]float64, NumPortfolios*4),
		StateHistoryDepth: 1,
		Seed:              0,
	}

	portfolioHighlight := &simulator.PartitionConfig{
		Name:      "portfolio_highlight",
		Iteration: &PortfolioHighlightIteration{},
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"policy_action": {Upstream: "policy_action"},
		},
		InitStateValues:   []float64{0, 0, 0, 0},
		StateHistoryDepth: 1,
		Seed:              0,
	}

	displayProgress := &simulator.PartitionConfig{
		Name:      "display_progress",
		Iteration: &DisplayProgressIteration{},
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"population_state": {Upstream: "population"},
		},
		InitStateValues:   []float64{0, 0},
		StateHistoryDepth: 1,
		Seed:              0,
	}

	displayOutcomes := &simulator.PartitionConfig{
		Name:      "display_outcomes",
		Iteration: &DisplayOutcomesIteration{},
		Params:    simulator.NewParams(map[string][]float64{}),
		ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
			"policy_action": {Upstream: "policy_action"},
		},
		InitStateValues:   []float64{0, 0, 0, 0},
		StateHistoryDepth: 1,
		Seed:              0,
	}

	gen := simulator.NewConfigGenerator()
	for _, p := range []*simulator.PartitionConfig{
		policyAction,
		population,
		cohort,
		stockTrajectory,
		cohortTrajectory,
		portfolioDots,
		portfolioHighlight,
		displayProgress,
		displayOutcomes,
	} {
		gen.SetPartition(p)
	}

	gen.SetSimulation(&simulator.SimulationConfig{
		OutputCondition: &simulator.EveryStepOutputCondition{},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: SimMonths,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: Stepsize},
		InitTimeValue:    0.0,
	})
	return gen
}
