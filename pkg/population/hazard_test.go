package population

import (
	"testing"

	"gonum.org/v1/gonum/floats/scalar"
)

func TestMonthlyHazardsReproduceCumulativeSurvival(t *testing.T) {
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	h := MonthlyHazardsFromCumulativeSurvival(surv)
	checkpoints := []int{12, 24, 36, 48, 60}
	for i, m := range checkpoints {
		got := CumulativeSurvivalAfterMonths(h, m)
		if !scalar.EqualWithinAbs(got, surv[i], 1e-9) {
			t.Fatalf("after %d months: got %.12f want %.12f", m, got, surv[i])
		}
	}
}
