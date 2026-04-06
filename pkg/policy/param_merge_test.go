package policy

import (
	"testing"

	"gonum.org/v1/gonum/floats/scalar"
)

func TestPortfolioParams_baselineNil(t *testing.T) {
	b := Portfolio{ID: "baseline"}
	if PortfolioParams(b) != nil {
		t.Fatal("expected nil")
	}
}

func TestPortfolioParams_reliefHasDeathScale(t *testing.T) {
	for _, p := range StandardPortfolios() {
		if p.ID != "rates_relief" {
			continue
		}
		m := PortfolioParams(p)
		if m == nil || !scalar.EqualWithinAbs(m["policy_death_hazard_scale"][0], 0.92, 1e-9) {
			t.Fatalf("%v", m)
		}
		if len(m["policy_sector_hazard_scale"]) != len(SectorOrder) {
			t.Fatal(len(m["policy_sector_hazard_scale"]))
		}
	}
}
