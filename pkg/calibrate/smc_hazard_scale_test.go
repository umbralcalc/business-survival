package calibrate

import (
	"math"
	"testing"

	"github.com/umbralcalc/stochadex/pkg/analysis"
	"gonum.org/v1/gonum/floats/scalar"
)

func TestSMC_HazardScaleRecoversUnity(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	cfg := SMCHazardScaleConfig{
		SurvivalFracs:   surv,
		Target5yr:       0.384,
		LikelihoodSigma: 0.025,
		NParticles:      24,
		NRounds:         3,
		PriorLo:         0.4,
		PriorHi:         1.8,
		ProposalSeed:    424242,
		Verbose:         testing.Verbose(),
	}
	applied, err := NewHazardScaleAppliedSMCInference(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result := analysis.RunSMCInference(applied)
	if result == nil {
		t.Fatal("RunSMCInference returned nil")
	}
	if !scalar.EqualWithinAbs(result.PosteriorMean[0], 1.0, 0.12) {
		t.Fatalf("posterior mean hazard_scale got %.4f want ~1.0", result.PosteriorMean[0])
	}
}

func TestRunSMCHazardScaleCalibration(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	mean, std, logMarg, err := RunSMCHazardScaleCalibration(SMCHazardScaleConfig{
		SurvivalFracs:   surv,
		Target5yr:       0.384,
		LikelihoodSigma: 0.025,
		NParticles:      16,
		NRounds:         2,
		PriorLo:         0.5,
		PriorHi:         1.5,
		ProposalSeed:    99,
	})
	if err != nil {
		t.Fatal(err)
	}
	if std <= 0 || math.IsNaN(logMarg) || math.IsInf(logMarg, 0) {
		t.Fatalf("std=%f logMarg=%f", std, logMarg)
	}
	if !scalar.EqualWithinAbs(mean, 1.0, 0.2) {
		t.Fatalf("mean=%f", mean)
	}
}
