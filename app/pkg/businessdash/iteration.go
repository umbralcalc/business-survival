package businessdash

import (
	_ "embed"
	"encoding/json"
	"math"
	"math/rand/v2"

	"github.com/umbralcalc/business-survival/pkg/policy"
	"github.com/umbralcalc/business-survival/pkg/population"
	"github.com/umbralcalc/stochadex/pkg/simulator"
	"gonum.org/v1/gonum/stat/distuv"
)

// SimMonths is the number of monthly outer steps before the dashboard
// halts. 120 = 10 years, matching the project's evaluate horizon.
const SimMonths = 120

// CohortMonths is the horizon over which the cohort sub-run tracks the
// decay of a 5000-business cohort planted at age 0. 60 months = 5 years,
// matching the project's "five-year cohort survival" metric.
const CohortMonths = 60

// Stepsize is the per-step time increment in months. The economic
// covariate index advances by floor(time) per step.
const Stepsize = 1.0

// Action vector layout. The slider/radio panel writes to
// action_state_values in this order.
const (
	PAIdxPortfolio  = 0
	PAIdxScenario   = 1
	PolicyActionLen = 2
)

// Portfolio indices — match StandardPortfolios() ordering in
// pkg/policy/portfolios.go.
const (
	PortfolioBaseline    = 0
	PortfolioRatesRelief = 1
	PortfolioStartup     = 2
	PortfolioIncubator   = 3
	PortfolioMentoring   = 4
	PortfolioBlend       = 5
	NumPortfolios        = 6
)

// Scenario indices — match policy.AllScenarioLabels ordering.
const (
	ScenarioBaseline  = 0
	ScenarioRecession = 1
	ScenarioExpansion = 2
	NumScenarios      = 3
)

// scenarioLabels lines up index → ScenarioLabel for the recipe lookup
// in the Leslie iteration. policy.AdjustCovariates expects the label.
var scenarioLabels = []policy.ScenarioLabel{
	policy.ScenarioBaseline,
	policy.ScenarioRecession,
	policy.ScenarioExpansion,
}

// ReferenceSurvival is the README's mean five-year cohort survival
// fraction at 64 replications, rows = portfolios, cols = scenarios.
// These are the anchor points for the comparison scatter and the
// readouts.
//
//	row: baseline / rates_relief / startup / incubator / mentoring / blend
//	col: baseline / recession / expansion
//
// Refresh by running:
//
//	go run ./cmd/evaluate -la E06000010 -runs 64 -months 120 \
//	    -out dat/evaluate_hull.json
//
// and copying the mean_cohort_5yr_survival_frac values from the JSON.
var ReferenceSurvival = [NumPortfolios][NumScenarios]float64{
	{0.3735, 0.3720, 0.3747}, // baseline
	{0.4139, 0.4124, 0.4140}, // rates_relief
	{0.3745, 0.3718, 0.3738}, // startup
	{0.3849, 0.3844, 0.3859}, // incubator
	{0.3982, 0.3943, 0.3983}, // mentoring
	{0.4133, 0.4113, 0.4147}, // blend
}

// ReferenceSurvivalStd is the README's std-dev across 64 replicate
// means for the five-year cohort survival fraction.
var ReferenceSurvivalStd = [NumPortfolios][NumScenarios]float64{
	{0.0068, 0.0069, 0.0067},
	{0.0071, 0.0069, 0.0077},
	{0.0073, 0.0072, 0.0072},
	{0.0058, 0.0067, 0.0065},
	{0.0059, 0.0066, 0.0079},
	{0.0070, 0.0073, 0.0064},
}

// ReferenceStock is the README's mean final register stock (12-sector
// aggregate at month 120) under the same configurations.
var ReferenceStock = [NumPortfolios][NumScenarios]float64{
	{5625.0, 5557.0, 5662.0},
	{6042.0, 5953.0, 6050.0},
	{6280.0, 6195.0, 6294.0},
	{6225.0, 6151.0, 6256.0},
	{5963.0, 5911.0, 6010.0},
	{6485.0, 6401.0, 6534.0},
}

// ReferenceStockStd matches the README's std-dev for final stock.
var ReferenceStockStd = [NumPortfolios][NumScenarios]float64{
	{72.0, 78.0, 81.0},
	{85.0, 72.0, 77.0},
	{76.0, 82.0, 77.0},
	{78.0, 77.0, 74.0},
	{80.0, 59.0, 71.0},
	{82.0, 72.0, 80.0},
}

// PortfolioLabels mirror policy.StandardPortfolios but trimmed to fit
// the small panel markers. Order matches Portfolio* constants.
var PortfolioLabels = []string{
	"none",
	"relief",
	"startup",
	"incubator",
	"mentoring",
	"blend",
}

// PortfolioFullLabels are the long names used in the radio buttons.
var PortfolioFullLabels = []string{
	"No additional intervention",
	"Rates & cash-flow relief",
	"Startup finance & first-year support",
	"Incubator / enterprise-zone style",
	"Mentoring & peer resilience",
	"Blended portfolio (relief + startup + mentoring)",
}

// ScenarioFullLabels are the long names used in the radio buttons.
var ScenarioFullLabels = []string{
	"Baseline macro path",
	"Recession overlay (+200 bps rates, +15% claimants)",
	"Expansion overlay (-100 bps rates, -8% claimants)",
}

// HullParams is the JSON shape produced by cmd/business/generate-data.
type HullParams struct {
	AreaCode              string    `json:"area_code"`
	AreaName              string    `json:"area_name"`
	SectorOrder           []string  `json:"sector_order"`
	SurvivalFracs         []float64 `json:"survival_fracs"`
	SectorHazardScales    []float64 `json:"sector_hazard_scales"`
	BaseBirthRates        []float64 `json:"base_birth_rates"`
	CovariateBankRates    []float64 `json:"covariate_bank_rates"`
	CovariateClaimants    []float64 `json:"covariate_claimants"`
	RateRef               float64   `json:"rate_ref"`
	ClaimantRef           float64   `json:"claimant_ref"`
	BirthElasticityRate   float64   `json:"birth_elasticity_rate"`
	BirthElasticityClaim  float64   `json:"birth_elasticity_claimant"`
	DeathElasticityRate   float64   `json:"death_elasticity_rate"`
	InitialStockPerSector []float64 `json:"initial_stock_per_sector"`
	CohortSize            float64   `json:"cohort_size"`
}

//go:embed data/hull_params.json
var embeddedParamsJSON []byte

var (
	hullParams       HullParams
	hullParamsLoaded bool
)

func loadHullParams() {
	if hullParamsLoaded {
		return
	}
	if err := json.Unmarshal(embeddedParamsJSON, &hullParams); err != nil {
		panic("businessdash: parse embedded params: " + err.Error())
	}
	hullParamsLoaded = true
}

// portfolioMultipliers describes the per-portfolio modifiers as a small
// struct cached at package init from policy.StandardPortfolios. Index
// matches the Portfolio* constants.
type portfolioMultipliers struct {
	BirthScale         float64
	DeathHazardScale   float64
	InfantHazardScale  float64
	SectorBirthScale   []float64 // length NSectors, 1 if absent
	SectorHazardScale  []float64
}

var portfolioTable []portfolioMultipliers

func ensurePortfolioTable() {
	if portfolioTable != nil {
		return
	}
	n := len(policy.SectorOrder)
	portfolios := policy.StandardPortfolios()
	portfolioTable = make([]portfolioMultipliers, len(portfolios))
	for i, p := range portfolios {
		m := portfolioMultipliers{
			BirthScale:        nonNeg(p.BirthScale, 1),
			DeathHazardScale:  nonNeg(p.DeathHazardScale, 1),
			InfantHazardScale: nonNeg(p.InfantHazardScale, 1),
			SectorBirthScale:  make([]float64, n),
			SectorHazardScale: make([]float64, n),
		}
		for s, name := range policy.SectorOrder {
			m.SectorBirthScale[s] = 1.0
			m.SectorHazardScale[s] = 1.0
			if v, ok := p.SectorBirthScale[name]; ok && v > 0 {
				m.SectorBirthScale[s] = v
			}
			if v, ok := p.SectorHazardScale[name]; ok && v > 0 {
				m.SectorHazardScale[s] = v
			}
		}
		portfolioTable[i] = m
	}
}

func nonNeg(v, def float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return def
	}
	return v
}

// terminated reports whether the dashboard has reached its outer horizon.
func terminated(timestepsHistory *simulator.CumulativeTimestepsHistory) bool {
	return timestepsHistory.Values.AtVec(0) >= float64(SimMonths)
}

// PolicyActionIteration is the slider/radio-driven action partition.
// It echoes the most recent action_state_values vector as state.
//
// State width: PolicyActionLen.
type PolicyActionIteration struct{}

func (p *PolicyActionIteration) Configure(int, *simulator.Settings) {}

func (p *PolicyActionIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	if terminated(timestepsHistory) {
		return stateHistories[partitionIndex].CopyStateRow(0)
	}
	out := make([]float64, PolicyActionLen)
	if actions, ok := params.GetOk("action_state_values"); ok {
		for i := 0; i < PolicyActionLen && i < len(actions); i++ {
			out[i] = actions[i]
		}
		return out
	}
	prev := stateHistories[partitionIndex].CopyStateRow(0)
	copy(out, prev[:PolicyActionLen])
	return out
}

// LesliePopulationIteration is the per-step Leslie-style update for the
// live monthly business population. It mirrors
// population.SingleLAPopulationIteration but takes the policy and
// scenario from an upstream action partition rather than fixed params,
// and reads its background calibration (survival, hazards, births,
// covariates, refs, elasticities) from the embedded Hull JSON.
//
// State width: NSectors * 60.
type LesliePopulationIteration struct {
	cohort bool // true for the cohort partition (no births)

	nSectors      int
	monthlyHazard [][]float64
	rates         []float64
	claimants     []float64

	rateRef    float64
	claimRef   float64
	eRate      float64
	eClaim     float64
	eDeath     float64
	baseBirth  []float64

	poisson  distuv.Poisson
	binomial distuv.Binomial
	src      rand.Source

	scratch []float64
}

// NewLesliePopulationIteration constructs the per-step Leslie iteration
// for either the live population (cohort=false) or the births-off
// cohort sub-run (cohort=true).
func NewLesliePopulationIteration(cohort bool) *LesliePopulationIteration {
	return &LesliePopulationIteration{cohort: cohort}
}

func (l *LesliePopulationIteration) Configure(
	partitionIndex int,
	settings *simulator.Settings,
) {
	loadHullParams()
	ensurePortfolioTable()
	is := settings.Iterations[partitionIndex]

	l.nSectors = len(hullParams.SectorOrder)
	if is.StateWidth != l.nSectors*60 {
		panic("businessdash: state width must equal nSectors*60")
	}

	base := population.MonthlyHazardsFromCumulativeSurvival(hullParams.SurvivalFracs)
	l.monthlyHazard = make([][]float64, l.nSectors)
	for sec := 0; sec < l.nSectors; sec++ {
		row := make([]float64, 60)
		for m := range base {
			h := base[m] * hullParams.SectorHazardScales[sec]
			if h < 0 {
				h = 0
			}
			if h > 1 {
				h = 1
			}
			row[m] = h
		}
		l.monthlyHazard[sec] = row
	}
	l.rates = hullParams.CovariateBankRates
	l.claimants = hullParams.CovariateClaimants
	l.rateRef = hullParams.RateRef
	l.claimRef = hullParams.ClaimantRef
	l.eRate = hullParams.BirthElasticityRate
	l.eClaim = hullParams.BirthElasticityClaim
	l.eDeath = hullParams.DeathElasticityRate
	l.baseBirth = hullParams.BaseBirthRates

	seed := is.Seed
	l.src = rand.NewPCG(seed, seed+1)
	l.poisson = distuv.Poisson{Lambda: 1.0, Src: l.src}
	l.binomial = distuv.Binomial{N: 1, P: 0.5, Src: l.src}

	l.scratch = make([]float64, is.StateWidth)
}

func pickSeries(xs []float64, t int) float64 {
	if len(xs) == 0 {
		return 0
	}
	if t < 0 {
		t = 0
	}
	if t >= len(xs) {
		return xs[len(xs)-1]
	}
	return xs[t]
}

func (l *LesliePopulationIteration) economicMultipliers(
	rate, claim float64,
) (birthMult, deathMult float64) {
	birthMult = math.Exp(
		l.eRate*(rate-l.rateRef) +
			l.eClaim*math.Log(claim/l.claimRef),
	)
	deathMult = math.Exp(l.eDeath * (rate - l.rateRef))
	if birthMult <= 0 || math.IsNaN(birthMult) {
		birthMult = 1
	}
	if deathMult <= 0 || math.IsNaN(deathMult) {
		deathMult = 1
	}
	return birthMult, deathMult
}

// scenarioAdjust returns the macro-overlay-adjusted (rate, claim) pair
// for the current step. policy.AdjustCovariates does the same thing on
// whole series but for live work we only need the per-step value.
func scenarioAdjust(rate, claim float64, scenario int) (float64, float64) {
	switch scenario {
	case ScenarioRecession:
		return rate + 0.02, claim * 1.15
	case ScenarioExpansion:
		return rate - 0.01, claim * 0.92
	default:
		return rate, claim
	}
}

func (l *LesliePopulationIteration) offset(sec, age int) int {
	return sec*60 + age
}

func (l *LesliePopulationIteration) poissonSample(lambda float64) float64 {
	if lambda <= 0 {
		return 0
	}
	l.poisson.Lambda = lambda
	return l.poisson.Rand()
}

func (l *LesliePopulationIteration) binomialSample(nFloat, prob float64) float64 {
	n := int(math.Round(nFloat))
	if n <= 0 {
		return 0
	}
	if prob <= 0 {
		return 0
	}
	if prob >= 1 {
		return float64(n)
	}
	l.binomial.N = float64(n)
	l.binomial.P = prob
	return l.binomial.Rand()
}

func (l *LesliePopulationIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	hist := stateHistories[partitionIndex]
	if terminated(timestepsHistory) {
		return hist.CopyStateRow(0)
	}

	// In the cohort sub-run, freeze state after CohortMonths.
	if l.cohort && timestepsHistory.Values.AtVec(0) >= float64(CohortMonths) {
		return hist.CopyStateRow(0)
	}

	action := params.Get("policy_action")
	portfolio := clampInt(int(math.Round(action[PAIdxPortfolio])), 0, NumPortfolios-1)
	scenario := clampInt(int(math.Round(action[PAIdxScenario])), 0, NumScenarios-1)
	mult := portfolioTable[portfolio]

	t := int(timestepsHistory.Values.AtVec(0))
	rate := pickSeries(l.rates, t)
	claim := pickSeries(l.claimants, t)
	rate, claim = scenarioAdjust(rate, claim, scenario)
	birthMult, deathMult := l.economicMultipliers(rate, claim)

	// Reset scratch.
	for i := range l.scratch {
		l.scratch[i] = 0
	}

	for sec := 0; sec < l.nSectors; sec++ {
		secBirth := mult.SectorBirthScale[sec]
		secHazard := mult.SectorHazardScale[sec]
		// New entrants at age 0. The cohort sub-run has births off so
		// the existing cohort planted at age 0 in the initial state
		// just decays.
		if !l.cohort {
			lambda := l.baseBirth[sec] * birthMult * mult.BirthScale * secBirth
			l.scratch[l.offset(sec, 0)] = l.poissonSample(lambda)
		}

		// Age 1..58: inflow from age-1 bucket.
		for age := 1; age <= 58; age++ {
			prev := hist.Values.At(0, l.offset(sec, age-1))
			h := l.monthlyHazard[sec][age-1] * deathMult * mult.DeathHazardScale * secHazard
			if age == 1 {
				h *= mult.InfantHazardScale
			}
			if h < 0 {
				h = 0
			}
			if h > 1 {
				h = 1
			}
			moved := l.binomialSample(prev, 1.0-h)
			l.scratch[l.offset(sec, age)] += moved
		}

		// Top bucket: inflow from 58 plus survivors already in 59.
		h58 := l.monthlyHazard[sec][58] * deathMult * mult.DeathHazardScale * secHazard
		if h58 < 0 {
			h58 = 0
		}
		if h58 > 1 {
			h58 = 1
		}
		prev58 := hist.Values.At(0, l.offset(sec, 58))
		into59 := l.binomialSample(prev58, 1.0-h58)

		h59 := l.monthlyHazard[sec][59] * deathMult * mult.DeathHazardScale * secHazard
		if h59 < 0 {
			h59 = 0
		}
		if h59 > 1 {
			h59 = 1
		}
		prev59 := hist.Values.At(0, l.offset(sec, 59))
		stay := l.binomialSample(prev59, 1.0-h59)
		l.scratch[l.offset(sec, 59)] = into59 + stay
	}

	out := make([]float64, len(l.scratch))
	copy(out, l.scratch)
	return out
}

// StockTrajectoryIteration projects the population partition's state
// down to a single scalar — the total stock count — so an
// AddLineChart bound to this partition picks up the trajectory from
// state[0].
//
// State width: 1.
type StockTrajectoryIteration struct{}

func (s *StockTrajectoryIteration) Configure(int, *simulator.Settings) {}

func (s *StockTrajectoryIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	if terminated(timestepsHistory) {
		return stateHistories[partitionIndex].CopyStateRow(0)
	}
	pop := params.Get("population_state")
	sum := 0.0
	for _, v := range pop {
		sum += v
	}
	return []float64{sum}
}

// CohortTrajectoryIteration projects the cohort partition's state
// down to the fraction surviving (sum / initial cohort size).
//
// State width: 1.
type CohortTrajectoryIteration struct{}

func (c *CohortTrajectoryIteration) Configure(int, *simulator.Settings) {}

func (c *CohortTrajectoryIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	prev := stateHistories[partitionIndex].CopyStateRow(0)
	if terminated(timestepsHistory) {
		return prev
	}
	loadHullParams()
	// After CohortMonths the cohort iteration freezes its state, so
	// the fraction would otherwise drift up due to noisy zero births;
	// hold the last observed value instead.
	if timestepsHistory.Values.AtVec(0) >= float64(CohortMonths) {
		return prev
	}
	pop := params.Get("cohort_state")
	sum := 0.0
	for _, v := range pop {
		sum += v
	}
	frac := 0.0
	if hullParams.CohortSize > 0 {
		frac = sum / hullParams.CohortSize
	}
	return []float64{frac}
}

// Canvas geometry — shared by businessdash.go (renderer placement) and
// the scatter/marker iterations below.
const (
	CanvasWidth  = 640
	CanvasHeight = 480

	// Portfolio comparison scatter (top half).
	ScatterX      = 130
	ScatterY      = 60
	ScatterWidth  = 420
	ScatterHeight = 150

	// Scatter axis range. X = five-year cohort survival fraction; the
	// observed reference values sit in [0.37, 0.42] for Hull, so widen
	// to [0.34, 0.44] to give the markers a comfortable margin.
	ScatterMinSurvival = 0.34
	ScatterMaxSurvival = 0.44
	// Y = mean final register stock. Reference values span ~5500–6500,
	// so widen to [5300, 6700].
	ScatterMinStock = 5300.0
	ScatterMaxStock = 6700.0

	// Stock trajectory (bottom-left). Starts below the scatter's
	// x-axis caption so the title row doesn't collide with the
	// scatter's "5-yr cohort survival" caption.
	StockX      = 70
	StockY      = 310
	StockWidth  = 220
	StockHeight = 120

	// Cohort trajectory (bottom-right).
	CohortX      = 380
	CohortY      = 310
	CohortWidth  = 220
	CohortHeight = 120

	// Markers for the scatter.
	MarkerSize    = 9
	HighlightSize = 13
)

// PortfolioDotsIteration emits one dot per portfolio on the scatter
// plot, positioned at the reference (survival, stock) point for the
// currently-selected scenario.
//
// State width: NumPortfolios * 4 floats (x, y, w, h per marker).
type PortfolioDotsIteration struct{}

func (p *PortfolioDotsIteration) Configure(int, *simulator.Settings) {}

func (p *PortfolioDotsIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	action := params.Get("policy_action")
	scenario := clampInt(int(math.Round(action[PAIdxScenario])), 0, NumScenarios-1)
	out := make([]float64, NumPortfolios*4)
	for i := 0; i < NumPortfolios; i++ {
		x := survivalToX(ReferenceSurvival[i][scenario])
		y := stockToY(ReferenceStock[i][scenario])
		out[i*4+0] = x - float64(MarkerSize)/2
		out[i*4+1] = y - float64(MarkerSize)/2
		out[i*4+2] = float64(MarkerSize)
		out[i*4+3] = float64(MarkerSize)
	}
	return out
}

// PortfolioHighlightIteration emits a larger marker at the user's
// currently-selected portfolio's (survival, stock) point.
//
// State width: 4.
type PortfolioHighlightIteration struct{}

func (p *PortfolioHighlightIteration) Configure(int, *simulator.Settings) {}

func (p *PortfolioHighlightIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	action := params.Get("policy_action")
	portfolio := clampInt(int(math.Round(action[PAIdxPortfolio])), 0, NumPortfolios-1)
	scenario := clampInt(int(math.Round(action[PAIdxScenario])), 0, NumScenarios-1)
	x := survivalToX(ReferenceSurvival[portfolio][scenario])
	y := stockToY(ReferenceStock[portfolio][scenario])
	return []float64{
		x - float64(HighlightSize)/2,
		y - float64(HighlightSize)/2,
		float64(HighlightSize),
		float64(HighlightSize),
	}
}

// DisplayProgressIteration surfaces the simulation's month count and
// the live total stock for the inline readout under the scatter panel.
//
// State width: 2.
type DisplayProgressIteration struct{}

func (d *DisplayProgressIteration) Configure(int, *simulator.Settings) {}

func (d *DisplayProgressIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	if terminated(timestepsHistory) {
		return stateHistories[partitionIndex].CopyStateRow(0)
	}
	pop := params.Get("population_state")
	sum := 0.0
	for _, v := range pop {
		sum += v
	}
	month := timestepsHistory.Values.AtVec(0)
	return []float64{month, sum}
}

// DisplayOutcomesIteration surfaces the README's reference five-year
// cohort survival % and final stock under the currently-selected
// portfolio + scenario combination. These are the "official"
// 64-replication numbers from dat/evaluate_hull.json, anchored against
// the live trajectory so the reader can see how their live single-draw
// stack against the offline mean ± std-dev.
//
// State width: 4 — [survival%, survival_std%, stock, stock_std].
type DisplayOutcomesIteration struct{}

func (d *DisplayOutcomesIteration) Configure(int, *simulator.Settings) {}

func (d *DisplayOutcomesIteration) Iterate(
	params *simulator.Params,
	partitionIndex int,
	stateHistories []*simulator.StateHistory,
	timestepsHistory *simulator.CumulativeTimestepsHistory,
) []float64 {
	action := params.Get("policy_action")
	portfolio := clampInt(int(math.Round(action[PAIdxPortfolio])), 0, NumPortfolios-1)
	scenario := clampInt(int(math.Round(action[PAIdxScenario])), 0, NumScenarios-1)
	survPct := ReferenceSurvival[portfolio][scenario] * 100
	survStdPct := ReferenceSurvivalStd[portfolio][scenario] * 100
	stock := ReferenceStock[portfolio][scenario]
	stockStd := ReferenceStockStd[portfolio][scenario]
	return []float64{survPct, survStdPct, stock, stockStd}
}

// Coordinate-system helpers for the scatter. The trajectory panels are
// auto-scaled by dexetera's AddLineChart so they don't need helpers.

func survivalToX(frac float64) float64 {
	if frac < ScatterMinSurvival {
		frac = ScatterMinSurvival
	}
	if frac > ScatterMaxSurvival {
		frac = ScatterMaxSurvival
	}
	f := (frac - ScatterMinSurvival) / (ScatterMaxSurvival - ScatterMinSurvival)
	return float64(ScatterX) + f*float64(ScatterWidth)
}

func stockToY(stock float64) float64 {
	if stock < ScatterMinStock {
		stock = ScatterMinStock
	}
	if stock > ScatterMaxStock {
		stock = ScatterMaxStock
	}
	f := (stock - ScatterMinStock) / (ScatterMaxStock - ScatterMinStock)
	return float64(ScatterY) + float64(ScatterHeight) - f*float64(ScatterHeight)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
