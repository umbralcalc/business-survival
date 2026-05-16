// generate-data extracts the Hull-specific simulation parameters from
// the project's dat/ files into a small JSON file embedded by the
// businessdash WASM widget. Re-run when the underlying ONS, Companies
// House, or panel data is refreshed.
//
//	cd app && go run ./cmd/business/generate-data
//
// Output: app/pkg/businessdash/data/hull_params.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/umbralcalc/business-survival/pkg/calibrate"
	"github.com/umbralcalc/business-survival/pkg/geo"
	"github.com/umbralcalc/business-survival/pkg/policy"
	"github.com/umbralcalc/business-survival/pkg/population"
)

const (
	areaCode      = "E06000010"
	cohortYear    = 2019
	birthLookback = 36
	tailMonths    = 120
)

type hullParams struct {
	AreaCode               string    `json:"area_code"`
	AreaName               string    `json:"area_name"`
	SectorOrder            []string  `json:"sector_order"`
	SurvivalFracs          []float64 `json:"survival_fracs"`
	SectorHazardScales     []float64 `json:"sector_hazard_scales"`
	BaseBirthRates         []float64 `json:"base_birth_rates"`
	CovariateBankRates     []float64 `json:"covariate_bank_rates"`
	CovariateClaimants     []float64 `json:"covariate_claimants"`
	RateRef                float64   `json:"rate_ref"`
	ClaimantRef            float64   `json:"claimant_ref"`
	BirthElasticityRate    float64   `json:"birth_elasticity_rate"`
	BirthElasticityClaim   float64   `json:"birth_elasticity_claimant"`
	DeathElasticityRate    float64   `json:"death_elasticity_rate"`
	InitialStockPerSector  []float64 `json:"initial_stock_per_sector"`
	CohortSize             float64   `json:"cohort_size"`
}

type panelDoc struct {
	Authorities []struct {
		AreaCode string                `json:"area_code"`
		Rows     []calibrate.PanelRow  `json:"rows"`
	} `json:"authorities"`
}

type birthFile struct {
	Authorities []struct {
		AreaCode string `json:"area_code"`
		AreaName string `json:"area_name"`
	} `json:"authorities"`
}

func mustRel(dat string) string {
	// Resolve "../dat/<name>" or "dat/<name>" so the tool can be invoked
	// from either the repo root or the app/ subdir.
	for _, p := range []string{"../dat/" + dat, "dat/" + dat} {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	fmt.Fprintf(os.Stderr, "generate-data: cannot locate %s under ../dat or dat\n", dat)
	os.Exit(1)
	return ""
}

func main() {
	panelPath := mustRel("la_panel.json")
	birthsPath := mustRel("la_births.json")
	onsPath := mustRel("ons_demography.json")

	rawPanel, err := os.ReadFile(panelPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read panel:", err)
		os.Exit(1)
	}
	var pdoc panelDoc
	if err := json.Unmarshal(rawPanel, &pdoc); err != nil {
		fmt.Fprintln(os.Stderr, "unmarshal panel:", err)
		os.Exit(1)
	}
	var rows []calibrate.PanelRow
	for i := range pdoc.Authorities {
		if pdoc.Authorities[i].AreaCode == areaCode {
			rows = pdoc.Authorities[i].Rows
			break
		}
	}
	if len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "no panel rows for", areaCode)
		os.Exit(1)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Month < rows[j].Month })
	if len(rows) > tailMonths {
		rows = rows[len(rows)-tailMonths:]
	}

	rates := make([]float64, len(rows))
	claimants := make([]float64, len(rows))
	var rateSum, claimSum float64
	for i, r := range rows {
		rates[i] = r.BankRate
		claimants[i] = float64(r.ClaimantCount)
		rateSum += r.BankRate
		claimSum += float64(r.ClaimantCount)
	}
	rateRef := rateSum / float64(len(rows))
	claimRef := claimSum / float64(len(rows))

	surv, err := population.LoadSurvivalFracsFromONSJSON(onsPath, areaCode, cohortYear)
	if err != nil {
		fmt.Fprintln(os.Stderr, "survival fracs:", err)
		os.Exit(1)
	}

	// Read birth lambdas from la_births.json via the package helper.
	auth, err := readBirths(birthsPath, areaCode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "birth lookup:", err)
		os.Exit(1)
	}
	baseBirth := sectorBirthLambdas(auth.MonthlyBirths, birthLookback)

	nSec := len(policy.SectorOrder)
	scales := make([]float64, nSec)
	for i, name := range policy.SectorOrder {
		r := calibrate.DefaultSectorHazardRelatives[name]
		if r <= 0 {
			r = 1.0
		}
		scales[i] = r
	}
	mix := make(map[string]float64)
	var mixSum float64
	for i, name := range policy.SectorOrder {
		mix[name] = baseBirth[i]
		mixSum += baseBirth[i]
	}
	if mixSum > 0 {
		for k := range mix {
			mix[k] /= mixSum
		}
	}
	globalG := calibrate.FitGlobalHazardScale(
		surv, mix, calibrate.DefaultSectorHazardRelatives, surv[4],
	)
	for i := range scales {
		scales[i] *= globalG
	}

	mb, _ := geo.MeanMonthlyTotalBirths(birthsPath, areaCode, birthLookback)
	eRate, eClaim, eDeath, err := calibrate.SimulationElasticitiesFromPanel(panelPath, mb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "elasticities:", err)
		os.Exit(1)
	}

	// Seed an initial stock per sector so the live trajectory starts at
	// roughly the Hull register's mean level instead of zero — the
	// downstream Leslie iteration takes ~12 months to fill its ageing
	// pyramid from a cold start. Pick a per-sector total of ~initial
	// monthly births × 60 (saturation across age buckets at hazard ≈ 0)
	// scaled down by a survival factor matching the calibration. The
	// dashboard runtime evenly distributes this across the 60 age
	// buckets at startup.
	avgSurv := 0.5
	if surv[4] > 0 {
		avgSurv = (1.0 + surv[4]) / 2.0
	}
	initStock := make([]float64, nSec)
	for i := range initStock {
		initStock[i] = baseBirth[i] * 60.0 * avgSurv
	}

	// Cohort size matching the project's default for the survival sub-run.
	cohortSize := 5000.0

	out := hullParams{
		AreaCode:              areaCode,
		AreaName:              auth.AreaName,
		SectorOrder:           append([]string(nil), policy.SectorOrder...),
		SurvivalFracs:         surv,
		SectorHazardScales:    scales,
		BaseBirthRates:        baseBirth,
		CovariateBankRates:    rates,
		CovariateClaimants:    claimants,
		RateRef:               rateRef,
		ClaimantRef:           claimRef,
		BirthElasticityRate:   eRate,
		BirthElasticityClaim:  eClaim,
		DeathElasticityRate:   eDeath,
		InitialStockPerSector: initStock,
		CohortSize:            cohortSize,
	}

	outPath := filepath.Join("pkg", "businessdash", "data", "hull_params.json")
	if _, err := os.Stat("pkg"); err != nil {
		// Allow running from app/.
		outPath = filepath.Join("pkg", "businessdash", "data", "hull_params.json")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	buf, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, buf, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", outPath)
}

type birthMonthRow struct {
	Month    string         `json:"month"`
	Total    int            `json:"total"`
	BySector map[string]int `json:"by_sector"`
}

type birthAuthority struct {
	AreaCode      string          `json:"area_code"`
	AreaName      string          `json:"area_name"`
	MonthlyBirths []birthMonthRow `json:"monthly_births"`
}

type birthsDoc struct {
	Authorities []birthAuthority `json:"authorities"`
}

func readBirths(path, code string) (*birthAuthority, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc birthsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for i := range doc.Authorities {
		if doc.Authorities[i].AreaCode == code {
			sort.Slice(doc.Authorities[i].MonthlyBirths, func(a, b int) bool {
				return doc.Authorities[i].MonthlyBirths[a].Month < doc.Authorities[i].MonthlyBirths[b].Month
			})
			return &doc.Authorities[i], nil
		}
	}
	return nil, fmt.Errorf("no births for %s", code)
}

func sectorBirthLambdas(rows []birthMonthRow, lastN int) []float64 {
	out := make([]float64, len(policy.SectorOrder))
	if len(rows) == 0 {
		for i := range out {
			out[i] = 1.0
		}
		return out
	}
	if lastN > len(rows) {
		lastN = len(rows)
	}
	tail := rows[len(rows)-lastN:]
	sums := make(map[string]float64)
	for _, r := range tail {
		for s, c := range r.BySector {
			sums[s] += float64(c)
		}
	}
	n := float64(len(tail))
	for i, name := range policy.SectorOrder {
		v := sums[name] / n
		if v < 0.02 {
			v = 0.02
		}
		out[i] = v
	}
	return out
}
