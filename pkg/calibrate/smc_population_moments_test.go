package calibrate

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/umbralcalc/business-survival/pkg/population"
	"github.com/umbralcalc/stochadex/pkg/analysis"
	"github.com/umbralcalc/stochadex/pkg/simulator"
	"gonum.org/v1/gonum/floats/scalar"
)

func repoDat(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "dat", "ons_demography.json")
}

func TestSMCPopulationMoments_RecoversNearUnity(t *testing.T) {
	path := repoDat(t)
	surv, err := population.LoadSurvivalFracsFromONSJSON(path, "K02000001", 2019)
	if err != nil {
		t.Fatal(err)
	}
	baseBirth := 1200.0 / 60.0
	// Forward truth
	mMap := map[string][]float64{
		"survival_fracs":         surv,
		"base_birth_rate_scalar": {baseBirth},
		"param_values":           {1.0, 1.0},
	}
	mset := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Params:            simulator.Params{Map: mMap},
				InitStateValues:   []float64{0, 0},
				StateWidth:        2,
				StateHistoryDepth: 2,
				Seed:              99,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	mset.Init()
	mom := &population.PopulationSurvivalBirthMomentsIteration{}
	mom.Configure(0, mset)
	truth := mom.Iterate(&simulator.Params{Map: mMap}, 0, nil, nil)

	applied, err := NewPopulationMomentsAppliedSMCInference(SMCPopulationMomentsConfig{
		SurvivalFracs:              surv,
		Target5yr:                  truth[0],
		TargetMeanMonthlyBirths:    truth[1],
		BaseBirthRateScalar:        baseBirth,
		Sigma5yr:                   0.04,
		SigmaBirths:                15,
		NParticles:                 80,
		NRounds:                    4,
		HazardPriorLo:              0.4,
		HazardPriorHi:              2.0,
		BirthPriorLo:               0.4,
		BirthPriorHi:               2.0,
		ProposalSeed:               202,
		Verbose:                    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := analysis.RunSMCInference(applied)
	if result == nil {
		t.Fatal("nil SMC result")
	}
	if !scalar.EqualWithinAbs(result.PosteriorMean[0], 1.0, 0.25) {
		t.Logf("hazard posterior mean %v (want ~1)", result.PosteriorMean[0])
	}
	if !scalar.EqualWithinAbs(result.PosteriorMean[1], 1.0, 0.25) {
		t.Logf("birth posterior mean %v (want ~1)", result.PosteriorMean[1])
	}
}
