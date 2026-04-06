package calibrate

import (
	"testing"

	"github.com/umbralcalc/business-survival/pkg/population"
	"gonum.org/v1/gonum/floats/scalar"
)

func TestFitGlobalHazardScaleRecoversBlendedSurvival(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	target := population.CumulativeSurvivalAfterMonths(
		population.MonthlyHazardsFromCumulativeSurvival(surv),
		60,
	)
	mix := map[string]float64{
		"Professional": 0.5,
		"Hospitality":  0.5,
	}
	rel := map[string]float64{
		"Professional": 1.0,
		"Hospitality":  1.0,
	}
	g := FitGlobalHazardScale(surv, mix, rel, target)
	hz := BlendMonthlyHazard(
		population.MonthlyHazardsFromCumulativeSurvival(surv),
		mix,
		rel,
		g,
	)
	got := population.CumulativeSurvivalAfterMonths(hz, 60)
	if !scalar.EqualWithinAbs(got, target, 2e-3) {
		t.Fatalf("got %f want %f (scale=%f)", got, target, g)
	}
}
