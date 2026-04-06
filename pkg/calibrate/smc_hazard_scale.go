package calibrate

import (
	"fmt"

	"github.com/umbralcalc/stochadex/pkg/analysis"
	"github.com/umbralcalc/stochadex/pkg/inference"
	"github.com/umbralcalc/stochadex/pkg/simulator"
	"github.com/umbralcalc/business-survival/pkg/population"
)

// SMCHazardScaleConfig holds options for one-dimensional SMC over a global
// hazard multiplier applied to an ONS baseline survival curve.
type SMCHazardScaleConfig struct {
	// SurvivalFracs are five cumulative survival fractions (years 1–5), 0–1 scale.
	SurvivalFracs []float64
	// Target5yr is the observed / reference five-year survival to match.
	Target5yr float64
	// LikelihoodSigma is the observation standard deviation on the survival scale.
	LikelihoodSigma float64
	// NParticles is the SMC particle count (embedded inner width 2*NParticles).
	NParticles int
	// NRounds is the number of outer SMC steps (prior + NRounds-1 rejuvenations).
	NRounds int
	// PriorLo, PriorHi bound the uniform prior on the hazard multiplier.
	PriorLo, PriorHi float64
	// ProposalSeed seeds SMC proposals (embedded sim uses Seed+100 per stochadex analysis).
	ProposalSeed uint64
	// Verbose enables SMC printf diagnostics.
	Verbose bool
}

func validateSMCHazardScaleConfig(cfg SMCHazardScaleConfig) error {
	if len(cfg.SurvivalFracs) != 5 {
		return fmt.Errorf("calibrate: need 5 survival_fracs")
	}
	if cfg.NParticles < 2 {
		return fmt.Errorf("calibrate: need at least 2 particles")
	}
	if cfg.NRounds < 1 {
		return fmt.Errorf("calibrate: need at least 1 SMC round")
	}
	return nil
}

// NewHazardScaleAppliedSMCInference returns an AppliedSMCInference wired for
// stochadex/pkg/analysis.RunSMCInference using the hazard-scale inner model.
func NewHazardScaleAppliedSMCInference(cfg SMCHazardScaleConfig) (analysis.AppliedSMCInference, error) {
	if err := validateSMCHazardScaleConfig(cfg); err != nil {
		return analysis.AppliedSMCInference{}, err
	}
	sigma := cfg.LikelihoodSigma
	if sigma <= 0 {
		sigma = 0.03
	}
	varData := sigma * sigma
	target := cfg.Target5yr
	surv := append([]float64(nil), cfg.SurvivalFracs...)

	model := analysis.SMCParticleModel{
		Build: func(N, nParams int) *analysis.SMCInnerSimConfig {
			if nParams != 1 {
				panic("calibrate: hazard-scale SMC expects nParams == 1")
			}
			partitions := make([]*simulator.PartitionConfig, 0, 2*N)
			loglikeParts := make([]string, N)
			forwarding := make(map[string][]int)

			for p := 0; p < N; p++ {
				fwd := fmt.Sprintf("fwd_%d", p)
				ll := fmt.Sprintf("ll_%d", p)

				partitions = append(partitions, &simulator.PartitionConfig{
					Name:      fwd,
					Iteration: &population.ScaledCohortSurvivalIteration{},
					Params: simulator.NewParams(map[string][]float64{
						"survival_fracs": surv,
						"param_values":   {1.0},
					}),
					InitStateValues:   []float64{0},
					StateHistoryDepth: 2,
					Seed:              uint64(9000 + p),
				})

				partitions = append(partitions, &simulator.PartitionConfig{
					Name: ll,
					Iteration: &inference.DataComparisonIteration{
						Likelihood: &inference.NormalLikelihoodDistribution{
							AllowDefaultCovarianceFallback: true,
						},
					},
					Params: simulator.NewParams(map[string][]float64{
						"mean":               {0},
						"variance":           {varData},
						"default_covariance": {varData},
						"latest_data_values": {target},
						"cumulative":         {1},
						"burn_in_steps":      {0},
					}),
					ParamsFromUpstream: map[string]simulator.NamedUpstreamConfig{
						"mean": {Upstream: fwd},
					},
					InitStateValues:   []float64{0},
					StateHistoryDepth: 2,
					Seed:              uint64(9100 + p),
				})

				loglikeParts[p] = ll
				forwarding[fwd+"/param_values"] = []int{p * nParams}
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
		ProposalName:  "smc_proposals",
		SimName:       "smc_sim",
		PosteriorName: "smc_posterior",
		NumParticles:  cfg.NParticles,
		NumRounds:     cfg.NRounds,
		Priors: []inference.Prior{
			&inference.UniformPrior{Lo: cfg.PriorLo, Hi: cfg.PriorHi},
		},
		ParamNames: []string{"hazard_scale"},
		Model:      model,
		Seed:       cfg.ProposalSeed,
		Verbose:    cfg.Verbose,
	}, nil
}
