package calibrate

import (
	"fmt"

	"github.com/umbralcalc/stochadex/pkg/macros"
)

// RunSMCHazardScaleCalibration runs stochadex/pkg/macros.RunSMCInference for
// the hazard-scale model and returns posterior mean / std of the multiplier and
// log marginal likelihood.
func RunSMCHazardScaleCalibration(cfg SMCHazardScaleConfig) (mean, std, logMarg float64, err error) {
	applied, err := NewHazardScaleAppliedSMCInference(cfg)
	if err != nil {
		return 0, 0, 0, err
	}
	result := macros.RunSMCInference(applied)
	if result == nil {
		return 0, 0, 0, fmt.Errorf("calibrate: RunSMCInference returned nil (check particle / inner config)")
	}
	return result.PosteriorMean[0], result.PosteriorStd[0], result.LogMarginalLik, nil
}
