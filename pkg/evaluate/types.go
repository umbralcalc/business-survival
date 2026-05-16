package evaluate

import "github.com/umbralcalc/business-survival/pkg/policy"

// Row is one portfolio × scenario aggregate.
type Row struct {
	PortfolioID       string  `json:"portfolio_id"`
	PortfolioName     string  `json:"portfolio_name"`
	Scenario          string  `json:"scenario"`
	MeanFinalStock    float64 `json:"mean_final_stock"`
	StdFinalStock     float64 `json:"std_final_stock"`
	MeanCohort5yrFrac float64 `json:"mean_cohort_5yr_survival_frac"`
	StdCohort5yrFrac  float64 `json:"std_cohort_5yr_survival_frac"`
}

// Output is written by Run and batch drivers.
type Output struct {
	GeneratedAt string `json:"generated_at"`
	AreaCode    string `json:"area_code,omitempty"`
	AreaName    string `json:"area_name,omitempty"`
	Runs        int    `json:"runs"`
	StockMonths int    `json:"stock_months"`
	Rows        []Row  `json:"rows"`
}

// BatchOutput aggregates several LAs (e.g. TargetLAs sweep).
type BatchOutput struct {
	GeneratedAt string   `json:"generated_at"`
	Items       []Output `json:"items"`
}

// Config drives a single-LA evaluation run.
type Config struct {
	PanelPath    string
	BirthsPath   string
	OnsPath      string
	LACode       string
	CohortYear   int
	Runs         int
	StockMonths  int
	BirthLookback int
	CohortSize   float64
	Deterministic bool

	ERate, EClaim, EDeath, EGDP float64
	GDPIndexed                  bool

	AutoElasticities bool
	DisplacementLeak float64 // 0 = off

	DistressFromClaimants bool

	BootstrapPanels int // >0: resample with replacement panel row indices per replicate

	// PolicyJitter applies multiplicative U[1-f,1+f] noise to policy_* scalars per replicate (robustness).
	PolicyJitter float64

	Portfolios []policy.Portfolio // nil → StandardPortfolios()
}
