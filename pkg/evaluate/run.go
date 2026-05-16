package evaluate

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"
	"time"

	"github.com/umbralcalc/business-survival/pkg/calibrate"
	"github.com/umbralcalc/business-survival/pkg/geo"
	"github.com/umbralcalc/business-survival/pkg/policy"
	"github.com/umbralcalc/business-survival/pkg/population"
)

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

type birthFile struct {
	Authorities []birthAuthority `json:"authorities"`
}

func loadBirthAuthority(path, areaCode string) (*birthAuthority, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc birthFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for i := range doc.Authorities {
		if doc.Authorities[i].AreaCode == areaCode {
			return &doc.Authorities[i], nil
		}
	}
	return nil, fmt.Errorf("evaluate: no authority %s in %s", areaCode, path)
}

func sectorBirthLambdas(rows []birthMonthRow, lastN int) []float64 {
	if len(rows) == 0 {
		out := make([]float64, len(policy.SectorOrder))
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
	out := make([]float64, len(policy.SectorOrder))
	for i, name := range policy.SectorOrder {
		v := sums[name] / n
		if v < 0.02 {
			v = 0.02
		}
		out[i] = v
	}
	return out
}

func sumFloats(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v
	}
	return s
}

func cloneAndMerge(
	base map[string][]float64,
	sc policy.ScenarioLabel,
	pol policy.Portfolio,
	gdp []float64,
	rates []float64,
	claimants []float64,
) map[string][]float64 {
	r2, c2, g2 := policy.AdjustCovariates(rates, claimants, gdp, sc)
	p := make(map[string][]float64, len(base)+8)
	for k, v := range base {
		cp := make([]float64, len(v))
		copy(cp, v)
		p[k] = cp
	}
	p["covariate_bank_rates"] = r2
	p["covariate_claimants"] = c2
	if pol.ID != "baseline" {
		pp := policy.PortfolioParams(pol)
		for k, v := range pp {
			p[k] = v
		}
	}
	if len(g2) > 0 {
		p["covariate_gdp_growth"] = g2
	}
	return p
}

func initCohortState(width int, nAges int, perSectorCohort []float64) []float64 {
	s := make([]float64, width)
	for sec := range perSectorCohort {
		s[sec*nAges] = perSectorCohort[sec]
	}
	return s
}

func meanStd(samples []float64) (mean, std float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	for _, x := range samples {
		mean += x
	}
	mean /= float64(len(samples))
	if len(samples) < 2 {
		return mean, 0
	}
	var ss float64
	for _, x := range samples {
		d := x - mean
		ss += d * d
	}
	return mean, math.Sqrt(ss / float64(len(samples)-1))
}

func jitterPolicyParams(p map[string][]float64, rng *rand.Rand, frac float64) map[string][]float64 {
	if frac <= 0 {
		return p
	}
	out := make(map[string][]float64, len(p))
	for k, v := range p {
		cp := make([]float64, len(v))
		copy(cp, v)
		out[k] = cp
	}
	perturb := func(key string) {
		xs, ok := out[key]
		if !ok || len(xs) == 0 {
			return
		}
		u := 1 + (2*rng.Float64()-1)*frac
		if u < 0.1 {
			u = 0.1
		}
		if u > 5 {
			u = 5
		}
		xs[0] *= u
	}
	perturb("policy_birth_scale")
	perturb("policy_death_hazard_scale")
	perturb("policy_infant_hazard_scale")
	for _, key := range []string{"policy_sector_birth_scale", "policy_sector_hazard_scale"} {
		xs, ok := out[key]
		if !ok {
			continue
		}
		for i := range xs {
			u := 1 + (2*rng.Float64()-1)*frac
			if u < 0.1 {
				u = 0.1
			}
			xs[i] *= u
		}
	}
	return out
}

func resamplePanelSeries(rows []calibrate.PanelRow, rng *rand.Rand) (rates, claimants []float64) {
	n := len(rows)
	if n == 0 {
		return nil, nil
	}
	rates = make([]float64, n)
	claimants = make([]float64, n)
	for i := 0; i < n; i++ {
		j := rng.IntN(n)
		rates[i] = rows[j].BankRate
		claimants[i] = float64(rows[j].ClaimantCount)
	}
	return rates, claimants
}

// Run executes Monte Carlo evaluation for one LA.
func Run(cfg Config) (*Output, error) {
	if cfg.BirthLookback <= 0 {
		cfg.BirthLookback = 36
	}
	if cfg.CohortYear == 0 {
		cfg.CohortYear = 2019
	}
	if cfg.Runs <= 0 {
		cfg.Runs = 48
	}
	if cfg.StockMonths <= 0 {
		cfg.StockMonths = 120
	}
	if cfg.CohortSize <= 0 {
		cfg.CohortSize = 5000
	}

	var doc calibrate.PanelFile
	rawPanel, err := os.ReadFile(cfg.PanelPath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawPanel, &doc); err != nil {
		return nil, err
	}
	var laPanel *calibrate.LAPanel
	for i := range doc.Authorities {
		if doc.Authorities[i].AreaCode == cfg.LACode {
			laPanel = &doc.Authorities[i]
			break
		}
	}
	if laPanel == nil {
		return nil, fmt.Errorf("evaluate: LA %s not in panel", cfg.LACode)
	}
	rows := laPanel.Rows
	sort.Slice(rows, func(i, j int) bool { return rows[i].Month < rows[j].Month })

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

	eRate, eClaim, eDeath := cfg.ERate, cfg.EClaim, cfg.EDeath
	if cfg.AutoElasticities {
		mb, _ := geo.MeanMonthlyTotalBirths(cfg.BirthsPath, cfg.LACode, cfg.BirthLookback)
		er, ec, ed, err := calibrate.SimulationElasticitiesFromPanel(cfg.PanelPath, mb)
		if err == nil {
			eRate, eClaim, eDeath = er, ec, ed
		}
	}

	auth, err := loadBirthAuthority(cfg.BirthsPath, cfg.LACode)
	if err != nil {
		return nil, err
	}
	sort.Slice(auth.MonthlyBirths, func(i, j int) bool {
		return auth.MonthlyBirths[i].Month < auth.MonthlyBirths[j].Month
	})
	baseBirth := sectorBirthLambdas(auth.MonthlyBirths, cfg.BirthLookback)
	if cfg.DisplacementLeak > 0 {
		f := geo.DisplacementBirthFactor(cfg.LACode, cfg.BirthsPath, cfg.BirthLookback, cfg.DisplacementLeak)
		for i := range baseBirth {
			baseBirth[i] *= f
		}
	}

	surv, err := population.LoadSurvivalFracsFromONSJSON(cfg.OnsPath, cfg.LACode, cfg.CohortYear)
	if err != nil {
		return nil, err
	}

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

	gdp := []float64(nil)
	if cfg.GDPIndexed {
		gdp = make([]float64, len(rates))
		for i := range gdp {
			if i == 0 {
				gdp[i] = 2.0
				continue
			}
			dB := float64(rows[i].Births - rows[i-1].Births)
			dB /= math.Max(1, float64(rows[i-1].Births))
			gdp[i] = 2.0 + 100*dB
		}
	}

	baseParams := map[string][]float64{
		"survival_fracs":            surv,
		"sector_hazard_scales":      scales,
		"base_birth_rates":          baseBirth,
		"covariate_bank_rates":      rates,
		"covariate_claimants":       claimants,
		"rate_ref":                  {rateRef},
		"claimant_ref":              {claimRef},
		"birth_elasticity_rate":     {eRate},
		"birth_elasticity_claimant": {eClaim},
		"death_elasticity_rate":     {eDeath},
	}
	if cfg.GDPIndexed && len(gdp) > 0 {
		baseParams["covariate_gdp_growth"] = gdp
		baseParams["gdp_ref"] = []float64{2.0}
		baseParams["birth_elasticity_gdp"] = []float64{cfg.EGDP}
	}
	if cfg.Deterministic {
		baseParams["deterministic"] = []float64{1}
	}
	if cfg.DistressFromClaimants {
		boost := DistressHazardBoostFromClaimants(claimants, 8)
		baseParams["distress_hazard_boost"] = boost
	}

	width := nSec * 60
	initZero := make([]float64, width)

	portfolios := cfg.Portfolios
	if len(portfolios) == 0 {
		portfolios = policy.StandardPortfolios()
	}

	rng := rand.New(rand.NewPCG(42, uint64(time.Now().UnixNano())))

	var outRows []Row
	for pi, p := range portfolios {
		for si, sc := range policy.AllScenarioLabels {
			stockSamps := make([]float64, 0, cfg.Runs)
			cohortSamps := make([]float64, 0, cfg.Runs)
			for r := 0; r < cfg.Runs; r++ {
				bp := baseParams
				rts, clm := rates, claimants
				if cfg.BootstrapPanels > 0 {
					bp = make(map[string][]float64, len(baseParams)+2)
					for k, v := range baseParams {
						cp := make([]float64, len(v))
						copy(cp, v)
						bp[k] = cp
					}
					rts, clm = resamplePanelSeries(rows, rng)
					bp["covariate_bank_rates"] = rts
					bp["covariate_claimants"] = clm
					if cfg.DistressFromClaimants {
						bp["distress_hazard_boost"] = DistressHazardBoostFromClaimants(clm, 8)
					}
				}
				pMap := cloneAndMerge(bp, sc, p, gdp, rts, clm)
				if cfg.PolicyJitter > 0 && p.ID != "baseline" {
					pMap = jitterPolicyParams(pMap, rng, cfg.PolicyJitter)
				}
				seed := uint64(800_000 + pi*50_000 + si*3000 + r)
				final := population.RunToState(pMap, initZero, seed, cfg.StockMonths)
				if final == nil {
					return nil, fmt.Errorf("evaluate: empty simulation output")
				}
				stockSamps = append(stockSamps, sumFloats(final))

				lamSum := sumFloats(baseBirth)
				perSec := make([]float64, nSec)
				if lamSum > 0 {
					for i := range perSec {
						perSec[i] = cfg.CohortSize * baseBirth[i] / lamSum
					}
				} else {
					perSec[0] = cfg.CohortSize
				}
				cohort0 := initCohortState(width, 60, perSec)
				zeroBirths := make([]float64, nSec)
				pCohort := policy.MergeParamMaps(pMap, map[string][]float64{
					"base_birth_rates": zeroBirths,
				})
				cfinal := population.RunToState(pCohort, cohort0, seed+1_200_000, 60)
				if cfinal == nil {
					return nil, fmt.Errorf("evaluate: empty cohort output")
				}
				cohortSamps = append(cohortSamps, sumFloats(cfinal)/cfg.CohortSize)
			}
			ms, ss := meanStd(stockSamps)
			mc, scd := meanStd(cohortSamps)
			outRows = append(outRows, Row{
				PortfolioID:       p.ID,
				PortfolioName:     p.Name,
				Scenario:          string(sc),
				MeanFinalStock:    ms,
				StdFinalStock:     ss,
				MeanCohort5yrFrac: mc,
				StdCohort5yrFrac:  scd,
			})
		}
	}

	return &Output{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		AreaCode:    cfg.LACode,
		AreaName:    auth.AreaName,
		Runs:        cfg.Runs,
		StockMonths: cfg.StockMonths,
		Rows:        outRows,
	}, nil
}
