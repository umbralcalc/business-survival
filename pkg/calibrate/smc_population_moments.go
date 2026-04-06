package calibrate

import (
	"fmt"

	"github.com/umbralcalc/business-survival/pkg/population"
	"github.com/umbralcalc/stochadex/pkg/analysis"
	"github.com/umbralcalc/stochadex/pkg/inference"
	"github.com/umbralcalc/stochadex/pkg/simulator"
)

// SMCPopulationMomentsConfig configures 2D SMC over (hazard_multiplier,
// birth_multiplier) using a single-sector moments forward model and a
// bivariate Gaussian likelihood (stochadex inference.NormalLikelihoodDistribution).
type SMCPopulationMomentsConfig struct {
	SurvivalFracs []float64
	// Target5yr and TargetMeanMonthlyBirths are observation means y.
	Target5yr                 float64
	TargetMeanMonthlyBirths   float64
	BaseBirthRateScalar       float64
	Sigma5yr, SigmaBirths     float64
	NParticles, NRounds       int
	HazardPriorLo, HazardPriorHi float64
	BirthPriorLo, BirthPriorHi   float64
	ProposalSeed uint64
	Verbose      bool
}

func validateSMCPopulationMoments(cfg SMCPopulationMomentsConfig) error {
	if len(cfg.SurvivalFracs) != 5 {
		return fmt.Errorf("calibrate: need 5 survival_fracs")
	}
	if cfg.NParticles < 2 || cfg.NRounds < 1 {
		return fmt.Errorf("calibrate: SMC size invalid")
	}
	if cfg.BaseBirthRateScalar < 0 {
		return fmt.Errorf("calibrate: base birth rate must be non-negative")
	}
	return nil
}

// NewPopulationMomentsAppliedSMCInference builds analysis.AppliedSMCInference for the
// hazard × birth moments model (two uniform priors, 2×2 diagonal Gaussian likelihood).
func NewPopulationMomentsAppliedSMCInference(cfg SMCPopulationMomentsConfig) (analysis.AppliedSMCInference, error) {
	if err := validateSMCPopulationMoments(cfg); err != nil {
		return analysis.AppliedSMCInference{}, err
	}
	s1 := cfg.Sigma5yr
	if s1 <= 0 {
		s1 = 0.03
	}
	s2 := cfg.SigmaBirths
	if s2 <= 0 {
		s2 = 1.0
	}
	var1 := s1 * s1
	var2 := s2 * s2
	surv := append([]float64(nil), cfg.SurvivalFracs...)
	data := []float64{cfg.Target5yr, cfg.TargetMeanMonthlyBirths}
	targetCov := []float64{var1, 0, 0, var2} // 2×2 row-major for SymDense

	model := analysis.SMCParticleModel{
		Build: func(N, nParams int) *analysis.SMCInnerSimConfig {
			if nParams != 2 {
				panic("calibrate: population moments SMC expects nParams == 2")
			}
			partitions := make([]*simulator.PartitionConfig, 0, 2*N)
			loglikeParts := make([]string, N)
			forwarding := make(map[string][]int)
			for p := 0; p < N; p++ {
				fwd := fmt.Sprintf("mom_%d", p)
				ll := fmt.Sprintf("llm_%d", p)
				partitions = append(partitions, &simulator.PartitionConfig{
					Name:      fwd,
					Iteration: &population.PopulationSurvivalBirthMomentsIteration{},
					Params: simulator.NewParams(map[string][]float64{
						"survival_fracs":          surv,
						"base_birth_rate_scalar":  {cfg.BaseBirthRateScalar},
						"param_values":            {1.0, 1.0},
					}),
					InitStateValues:   []float64{0, 0},
					StateHistoryDepth: 2,
					Seed:              uint64(7000 + p),
				})
				partitions = append(partitions, &simulator.PartitionConfig{
					Name: ll,
					Iteration: &inference.DataComparisonIteration{
						Likelihood: &inference.NormalLikelihoodDistribution{
							AllowDefaultCovarianceFallback: true,
						},
					},
					Params: simulator.NewParams(map[string][]float64{
						"variance":           {var1, var2},
						"default_covariance": targetCov,
						"latest_data_values": data,
						"cumulative":         {1},
						"burn_in_steps":      {0},
					}),
					ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
						"mean": {Upstream: fwd},
					},
					InitStateValues:   []float64{0},
					StateHistoryDepth: 2,
					Seed:              uint64(7100 + p),
				})
				loglikeParts[p] = ll
				forwarding[fwd+"/param_values"] = []int{p*nParams + 0, p*nParams + 1}
			}
			return &analysis.SMCInnerSimConfig{
				Partitions: partitions,
				Simulation: &simulator.SimulationConfig{
					OutputCondition: &simulator.NilOutputCondition{},
					OutputFunction:  &simulator.NilOutputFunction{},
					TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
						MaxNumberOfSteps: 1,
					},
					TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1},
					InitTimeValue:    0,
				},
				LoglikePartitions: loglikeParts,
				ParamForwarding:   forwarding,
			}
		},
	}

	return analysis.AppliedSMCInference{
		ProposalName:  "smc_mom_proposals",
		SimName:       "smc_mom_sim",
		PosteriorName: "smc_mom_posterior",
		NumParticles:  cfg.NParticles,
		NumRounds:     cfg.NRounds,
		Priors: []inference.Prior{
			&inference.UniformPrior{Lo: cfg.HazardPriorLo, Hi: cfg.HazardPriorHi},
			&inference.UniformPrior{Lo: cfg.BirthPriorLo, Hi: cfg.BirthPriorHi},
		},
		ParamNames: []string{"hazard_scale", "birth_scale"},
		Model:      model,
		Seed:       cfg.ProposalSeed,
		Verbose:    cfg.Verbose,
	}, nil
}

// RunSMCPopulationMomentsCalibration runs 2-parameter SMC; returns posterior
// mean/std for hazard and birth scales plus log marginal likelihood.
func RunSMCPopulationMomentsCalibration(cfg SMCPopulationMomentsConfig) (hazardMean, hazardStd, birthMean, birthStd, logMarg float64, err error) {
	applied, err := NewPopulationMomentsAppliedSMCInference(cfg)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	result := analysis.RunSMCInference(applied)
	if result == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("calibrate: RunSMCInference returned nil")
	}
	if len(result.PosteriorMean) < 2 {
		return 0, 0, 0, 0, 0, fmt.Errorf("calibrate: expected 2 posterior means")
	}
	return result.PosteriorMean[0], result.PosteriorStd[0],
		result.PosteriorMean[1], result.PosteriorStd[1],
		result.LogMarginalLik, nil
}
