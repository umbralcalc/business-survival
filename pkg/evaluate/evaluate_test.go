package evaluate

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoDat(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "dat")
}

func TestRun_HullSmoke(t *testing.T) {
	root := repoDat(t)
	cfg := Config{
		PanelPath:     filepath.Join(root, "la_panel.json"),
		BirthsPath:    filepath.Join(root, "la_births.json"),
		OnsPath:       filepath.Join(root, "ons_demography.json"),
		LACode:        "E06000010",
		CohortYear:    2019,
		Runs:          2,
		StockMonths:   12,
		BirthLookback: 24,
		CohortSize:    2000,
		Deterministic: true,
	}
	out, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Rows) < 6 {
		t.Fatalf("rows: %d", len(out.Rows))
	}
	if out.Rows[0].MeanCohort5yrFrac <= 0 || out.Rows[0].MeanCohort5yrFrac > 1 {
		t.Fatalf("cohort survival out of range: %v", out.Rows[0].MeanCohort5yrFrac)
	}
}

func TestDistressSeriesNonNegative(t *testing.T) {
	cl := []float64{100, 102, 98, 110, 105, 108, 107}
	b := DistressHazardBoostFromClaimants(cl, 4)
	for _, v := range b {
		if v < 0 {
			t.Fatal(v)
		}
	}
}
