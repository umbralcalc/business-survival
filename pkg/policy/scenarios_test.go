package policy

import (
	"testing"

	"gonum.org/v1/gonum/floats/scalar"
)

func TestAdjustCovariates_RecessionRaisesStress(t *testing.T) {
	rates := []float64{0.04, 0.04}
	claim := []float64{1000.0, 1000.0}
	rr, cc, _ := AdjustCovariates(rates, claim, nil, ScenarioRecession)
	if rr[0] <= rates[0] || cc[0] <= claim[0] {
		t.Fatalf("recession overlay: rates=%v claim=%v", rr, cc)
	}
}

func TestAdjustCovariates_ExpansionLowersStress(t *testing.T) {
	rates := []float64{0.05, 0.05}
	claim := []float64{2000.0, 2000.0}
	rr, cc, _ := AdjustCovariates(rates, claim, nil, ScenarioExpansion)
	if rr[0] >= rates[0] || cc[0] >= claim[0] {
		t.Fatalf("expansion overlay: rates=%v claim=%v", rr, cc)
	}
}

func TestAdjustCovariates_BaselineIdentity(t *testing.T) {
	rates := []float64{0.03}
	claim := []float64{1500.0}
	rr, cc, gg := AdjustCovariates(rates, claim, []float64{2.0}, ScenarioBaseline)
	if !scalar.EqualWithinAbs(rr[0], rates[0], 1e-12) ||
		!scalar.EqualWithinAbs(cc[0], claim[0], 1e-12) ||
		!scalar.EqualWithinAbs(gg[0], 2.0, 1e-12) {
		t.Fatal(rr, cc, gg)
	}
}

func TestAdjustCovariates_RecessionGDPDown(t *testing.T) {
	_, _, gg := AdjustCovariates([]float64{0.04}, []float64{1000.0}, []float64{2.5, 2.5}, ScenarioRecession)
	if !(gg[0] < 2.5) {
		t.Fatal(gg)
	}
}
