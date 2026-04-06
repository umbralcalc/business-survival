package calibrate

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoRootDat(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "dat")
}

func TestPooledFirstDiffRegressionOnFixture(t *testing.T) {
	bRate, bC, n, err := PooledFirstDiffRegression(filepath.Join(repoRootDat(t), "la_panel.json"))
	if err != nil {
		t.Fatal(err)
	}
	if n < 100 {
		t.Fatalf("short panel n=%d", n)
	}
	if bRate == 0 && bC == 0 {
		t.Fatal("expected non-zero coefficients")
	}
	t.Logf("pooled FD: beta_rate=%.4f beta_log_claimant=%.4f n=%d", bRate, bC, n)
}

func TestCOVIDBirthSlumpFromLA_Births(t *testing.T) {
	monthly, err := NationalMonthlySectorTotals(filepath.Join(repoRootDat(t), "la_births.json"))
	if err != nil {
		t.Fatal(err)
	}
	h, tech := COVIDBirthSlump(monthly)
	if h == 0 || tech == 0 {
		t.Fatal("missing YoY ratios")
	}
	if h >= tech {
		t.Fatalf("expected hospitality slump stronger than technology in 2020-04 YoY; hosp=%f tech=%f", h, tech)
	}
}

func TestPanelRecessionWindows(t *testing.T) {
	y09, cov, rest, n09, nc, nr, err := PanelRecessionWindows(filepath.Join(repoRootDat(t), "la_panel.json"))
	if err != nil {
		t.Fatal(err)
	}
	if nc == 0 || nr == 0 {
		t.Fatalf("counts covid=%d rest=%d", nc, nr)
	}
	t.Logf("mean d_births: 2009=%.3f (n=%d) covid=%.3f other=%.3f", y09, n09, cov, rest)
}
