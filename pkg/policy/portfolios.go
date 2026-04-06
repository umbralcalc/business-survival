package policy

// EffectPrior is a literature-informed plausible range for a scalar multiplier
// (point portfolios use mid-range conservative values).
type EffectPrior struct {
	// Multiplier is applied to births (×) or hazards (×); 1 = no effect.
	Low, High float64
	Source    string
}

// Portfolio describes one intervention bundle as multipliers on the
// monthly Leslie process (see population.SingleLAPopulationIteration).
type Portfolio struct {
	ID        string
	Name      string
	Summary   string
	BudgetGBP float64 // indicative annual envelope for reporting only

	BirthScale         float64
	DeathHazardScale   float64
	InfantHazardScale  float64
	SectorBirthScale   map[string]float64 // optional per-sector birth multiplier
	SectorHazardScale  map[string]float64 // optional per-sector hazard multiplier
	PriorsBirth        *EffectPrior
	PriorsDeath        *EffectPrior
	PriorsInfantDeath  *EffectPrior
}

// LiteraturePriorsTable documents evaluation evidence (see README Week 7–8).
// Point estimates in StandardPortfolios sit inside these bands.
var LiteraturePriorsTable = []struct {
	Intervention string
	Outcome      string
	Prior        EffectPrior
}{
	{
		Intervention: "Small business rate relief / property tax relief",
		Outcome:      "Distress-related exit hazard",
		Prior: EffectPrior{
			Low: 0.88, High: 0.98,
			Source: "BEIS/DLUHC rates consultations; SBRR uptake evidence — cash relief reduces payment pressure (directional; causal ATEs mixed).",
		},
	},
	{
		Intervention: "Startup loans / seed grants (year 1)",
		Outcome:      "First-year survival / continuation",
		Prior: EffectPrior{
			Low: 0.90, High: 1.0,
			Source: "British Business Bank programme statistics; micro-evaluation literature on loan guarantees — small positive continuation effects.",
		},
	},
	{
		Intervention: "Place-based zones / incubators",
		Outcome:      "Local formation (birth) intensity",
		Prior: EffectPrior{
			Low: 1.02, High: 1.15,
			Source: "Centre for Cities Enterprise Zone synthesis (jobs/businesses with displacement caveats); UEZ evaluation (2025).",
		},
	},
	{
		Intervention: "Mentoring / managerial support",
		Outcome:      "Chronic failure hazard after infancy",
		Prior: EffectPrior{
			Low: 0.90, High: 0.97,
			Source: "BEIS business support meta-evaluations; mentoring RCTs in SMEs — modest hazard reductions.",
		},
	},
}

// StandardPortfolios returns baseline plus three actionable bundles plus a
// blended portfolio.
func StandardPortfolios() []Portfolio {
	return []Portfolio{
		{
			ID:      "baseline",
			Name:    "No additional intervention",
			Summary: "Calibrated demography + economics only.",
		},
		{
			ID:      "rates_relief",
			Name:    "Rates & cash-flow relief",
			Summary: "SBRR-style support tilted to high-fixed-cost sectors.",
			BudgetGBP:            4_500_000,
			BirthScale:           1.0,
			DeathHazardScale:     0.92,
			InfantHazardScale:    1.0,
			SectorHazardScale:    map[string]float64{"Hospitality": 0.88, "Retail": 0.94},
			PriorsDeath:          &LiteraturePriorsTable[0].Prior,
		},
		{
			ID:      "startup_grants",
			Name:    "Startup finance & first-year support",
			Summary: "Formation nudge + stronger first-year continuation.",
			BudgetGBP:           3_800_000,
			BirthScale:          1.10,
			DeathHazardScale:    1.0,
			InfantHazardScale:   0.90,
			SectorBirthScale:    map[string]float64{"Technology": 1.14, "Professional": 1.08},
			PriorsBirth:         &LiteraturePriorsTable[2].Prior,
			PriorsInfantDeath:   &LiteraturePriorsTable[1].Prior,
		},
		{
			ID:      "incubator_ez",
			Name:    "Incubator / enterprise-zone style",
			Summary: "Place-based bias to tradable services & hospitality.",
			BudgetGBP:         5_000_000,
			BirthScale:        1.06,
			DeathHazardScale:  0.97,
			SectorBirthScale: map[string]float64{
				"Technology": 1.12, "Hospitality": 1.10, "Retail": 1.06,
			},
			PriorsBirth:  &LiteraturePriorsTable[2].Prior,
			PriorsDeath:  &LiteraturePriorsTable[0].Prior,
		},
		{
			ID:      "mentoring_resilience",
			Name:    "Mentoring & peer resilience network",
			Summary: "Broad hazard moderation; extra help through first year.",
			BudgetGBP:          2_200_000,
			BirthScale:         1.02,
			DeathHazardScale:   0.94,
			InfantHazardScale:  0.93,
			PriorsDeath:        &LiteraturePriorsTable[3].Prior,
			PriorsInfantDeath:  &LiteraturePriorsTable[3].Prior,
		},
		{
			ID:      "blend_balanced",
			Name:    "Blended portfolio (relief + startup + mentoring)",
			Summary: "Illustrative split budget: ~40% relief weighted, 35% startup, 25% mentoring (effect composition).",
			BudgetGBP:          4_200_000,
			BirthScale:         1.07,
			DeathHazardScale:   0.91,
			InfantHazardScale:  0.91,
			SectorHazardScale:  map[string]float64{"Hospitality": 0.90},
			SectorBirthScale:   map[string]float64{"Technology": 1.10},
		},
	}
}
