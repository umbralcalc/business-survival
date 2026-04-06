package population

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/umbralcalc/stochadex/pkg/simulator"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/floats/scalar"
)

func configureIterations(settings *simulator.Settings, impl *simulator.Implementations) {
	for i := range impl.Iterations {
		impl.Iterations[i].Configure(i, settings)
	}
}

func repoRootDat(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "dat", "ons_demography.json")
}

func constCovariateParams(months int, rate, claimant float64) map[string][]float64 {
	rates := make([]float64, months)
	for i := range rates {
		rates[i] = rate
	}
	return map[string][]float64{
		"survival_fracs":            {0.946, 0.747, 0.559, 0.45, 0.384},
		"sector_hazard_scales":      {1},
		"base_birth_rates":          {0},
		"covariate_bank_rates":      rates,
		"covariate_claimants":       {claimant},
		"rate_ref":                  {rate},
		"claimant_ref":              {claimant},
		"birth_elasticity_rate":     {0},
		"birth_elasticity_claimant": {0},
		"death_elasticity_rate":     {0},
		"deterministic":             {1},
	}
}

func initCohortState(cohort float64) []float64 {
	s := make([]float64, 60)
	s[0] = cohort
	return s
}

func sumState(state []float64) float64 {
	sum := 0.0
	for _, v := range state {
		sum += v
	}
	return sum
}

func TestSingleLA_CohortSurvivalMatchesONSUK(t *testing.T) {
	path := repoRootDat(t)
	surv, err := LoadSurvivalFracsFromONSJSON(path, "K02000001", 2019)
	if err != nil {
		t.Fatal(err)
	}
	cohort := 10_000.0
	pMap := constCovariateParams(120, 0.5, 15000)
	pMap["survival_fracs"] = surv

	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   initCohortState(cohort),
				Seed:              1,
				StateWidth:        60,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()

	store := simulator.NewStateTimeStorage()
	iter := &SingleLAPopulationIteration{}
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{iter},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 60,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	configureIterations(settings, impl)
	coordinator := simulator.NewPartitionCoordinator(settings, impl)
	coordinator.Run()
	vals := store.GetValues("population")
	last := vals[len(vals)-1]
	got := sumState(last)
	want := cohort * surv[4]
	if !scalar.EqualWithinAbs(got, want, 1e-6) {
		t.Fatalf("5yr cohort survival: got %f want %f", got, want)
	}
}

func TestSingleLA_Hull2019_CohortFiveYearSurvival(t *testing.T) {
	path := repoRootDat(t)
	surv, err := LoadSurvivalFracsFromONSJSON(path, "E06000010", 2019)
	if err != nil {
		t.Fatal(err)
	}
	cohort := 5_000.0
	pMap := constCovariateParams(80, 0.5, 15000)
	pMap["survival_fracs"] = surv

	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   initCohortState(cohort),
				Seed:              2,
				StateWidth:        60,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()

	store := simulator.NewStateTimeStorage()
	iter := &SingleLAPopulationIteration{}
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{iter},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 60,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	configureIterations(settings, impl)
	coordinator := simulator.NewPartitionCoordinator(settings, impl)
	coordinator.Run()
	vals := store.GetValues("population")
	got := sumState(vals[len(vals)-1])
	want := cohort * surv[4]
	if !scalar.EqualWithinAbs(got, want, 1e-6) {
		t.Fatalf("Hull 5yr survival: got %f want %f", got, want)
	}
}

func runPopulationSteps(
	pMap map[string][]float64,
	cohort float64,
	seed uint64,
	steps int,
) []float64 {
	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   initCohortState(cohort),
				Seed:              seed,
				StateWidth:        60,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()
	store := simulator.NewStateTimeStorage()
	iter := &SingleLAPopulationIteration{}
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{iter},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: steps,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	configureIterations(settings, impl)
	coordinator := simulator.NewPartitionCoordinator(settings, impl)
	coordinator.Run()
	vals := store.GetValues("population")
	return vals[len(vals)-1]
}

func TestSingleLA_policyDeathScaleRaisesSurvival(t *testing.T) {
	path := repoRootDat(t)
	surv, err := LoadSurvivalFracsFromONSJSON(path, "K02000001", 2019)
	if err != nil {
		t.Fatal(err)
	}
	cohort := 10_000.0
	base := constCovariateParams(120, 0.5, 15000)
	base["survival_fracs"] = surv
	withPolicy := make(map[string][]float64)
	for k, v := range base {
		withPolicy[k] = v
	}
	withPolicy["policy_death_hazard_scale"] = []float64{0.85}

	lastBase := runPopulationSteps(base, cohort, 42, 60)
	lastPol := runPopulationSteps(withPolicy, cohort, 42, 60)
	if sumState(lastPol) <= sumState(lastBase) {
		t.Fatalf("policy lowering hazard should increase survivors: base=%f policy=%f",
			sumState(lastBase), sumState(lastPol))
	}
}

func TestSingleLA_distressBoostRaisesDeaths(t *testing.T) {
	base := constCovariateParams(120, 0.5, 15000)
	surv := []float64{0.946, 0.747, 0.559, 0.45, 0.384}
	base["survival_fracs"] = surv
	dist := make(map[string][]float64)
	for k, v := range base {
		dist[k] = v
	}
	boost := make([]float64, 120)
	for i := range boost {
		boost[i] = 0.15
	}
	dist["distress_hazard_boost"] = boost

	lastBase := runPopulationSteps(base, 10_000, 7, 60)
	lastDist := runPopulationSteps(dist, 10_000, 7, 60)
	if sumState(lastDist) >= sumState(lastBase) {
		t.Fatalf("distress should reduce survivors: base=%f dist=%f", sumState(lastBase), sumState(lastDist))
	}
}

func TestSingleLA_policyBirthScaleRaisesStock(t *testing.T) {
	base := constCovariateParams(36, 0.5, 10000)
	base["base_birth_rates"] = []float64{10}
	base["deterministic"] = []float64{1}
	withPolicy := make(map[string][]float64)
	for k, v := range base {
		withPolicy[k] = v
	}
	withPolicy["policy_birth_scale"] = []float64{1.2}

	lastBase := runPopulationSteps(base, 0, 1, 36)
	// cohort 0: only births accumulate
	lastPol := runPopulationSteps(withPolicy, 0, 1, 36)
	if sumState(lastPol) <= sumState(lastBase) {
		t.Fatalf("higher birth policy should raise stock base=%f pol=%f",
			sumState(lastBase), sumState(lastPol))
	}
}

func TestSingleLA_StochasticCohortMeanNearONS(t *testing.T) {
	path := repoRootDat(t)
	surv, err := LoadSurvivalFracsFromONSJSON(path, "K02000001", 2019)
	if err != nil {
		t.Fatal(err)
	}
	nRuns := 400
	cohort := 500.0
	pMap := constCovariateParams(120, 0.5, 15000)
	pMap["survival_fracs"] = surv
	delete(pMap, "deterministic")

	wantFrac := surv[4]
	var mean float64
	for r := 0; r < nRuns; r++ {
		last := runPopulationSteps(pMap, cohort, uint64(100+r), 60)
		mean += sumState(last) / cohort
	}
	mean /= float64(nRuns)
	if floats.Distance([]float64{mean}, []float64{wantFrac}, 2) > 0.02 {
		t.Fatalf("Monte Carlo mean survival frac got %f want ~%f", mean, wantFrac)
	}
}

func TestSingleLA_GDPCovariateRaisesBirthsWhenElasticityPositive(t *testing.T) {
	gdpHigh := make([]float64, 24)
	gdpLow := make([]float64, 24)
	for i := range gdpHigh {
		gdpHigh[i] = 3.0
		gdpLow[i] = -1.0
	}
	base := constCovariateParams(24, 0.1, 10000)
	base["deterministic"] = []float64{1}
	base["base_birth_rates"] = []float64{15}
	base["birth_elasticity_gdp"] = []float64{0.08}
	base["gdp_ref"] = []float64{0.0}
	base["covariate_gdp_growth"] = gdpLow
	pLow := make(map[string][]float64)
	pHigh := make(map[string][]float64)
	for k, v := range base {
		pLow[k] = v
		pHigh[k] = v
	}
	pHigh["covariate_gdp_growth"] = gdpHigh
	initEmpty := make([]float64, 60)

	runTotal := func(p map[string][]float64) float64 {
		settings := &simulator.Settings{
			Iterations: []simulator.IterationSettings{
				{
					Name:              "population",
					Params:            simulator.Params{Map: p},
					InitStateValues:   append([]float64(nil), initEmpty...),
					Seed:              21,
					StateWidth:        60,
					StateHistoryDepth: 2,
				},
			},
			InitTimeValue:         0,
			TimestepsHistoryDepth: 2,
		}
		settings.Init()
		store := simulator.NewStateTimeStorage()
		impl := &simulator.Implementations{
			Iterations:      []simulator.Iteration{&SingleLAPopulationIteration{}},
			OutputCondition: &simulator.EveryStepOutputCondition{},
			OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
			TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
				MaxNumberOfSteps: 24,
			},
			TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
		}
		configureIterations(settings, impl)
		simulator.NewPartitionCoordinator(settings, impl).Run()
		return sumState(store.GetValues("population")[len(store.GetValues("population"))-1])
	}
	if runTotal(pHigh) <= runTotal(pLow)+50 {
		t.Fatal("expected higher GDP growth path to increase stock with positive birth_elasticity_gdp")
	}
}

func TestSingleLA_EconomicSensitivityRaisesBirthsWithRate(t *testing.T) {
	lowRates := make([]float64, 48)
	highRates := make([]float64, 48)
	for i := range lowRates {
		lowRates[i] = 0.01
		highRates[i] = 0.2
	}
	base := constCovariateParams(48, 0.5, 10000)
	base["deterministic"] = []float64{1}
	base["base_birth_rates"] = []float64{20}
	base["birth_elasticity_rate"] = []float64{1.5}
	base["rate_ref"] = []float64{0.105}
	pLow := make(map[string][]float64)
	pHigh := make(map[string][]float64)
	for k, v := range base {
		pLow[k] = v
		pHigh[k] = v
	}
	pLow["covariate_bank_rates"] = lowRates
	pHigh["covariate_bank_rates"] = highRates

	initEmpty := make([]float64, 60)
	settingsLow := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pLow},
				InitStateValues:   initEmpty,
				Seed:              9,
				StateWidth:        60,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settingsLow.Init()
	storeLow := simulator.NewStateTimeStorage()
	implLow := &simulator.Implementations{
		Iterations:      []simulator.Iteration{&SingleLAPopulationIteration{}},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: storeLow},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 36,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	configureIterations(settingsLow, implLow)
	coordinator := simulator.NewPartitionCoordinator(settingsLow, implLow)
	coordinator.Run()
	totalLow := sumState(storeLow.GetValues("population")[len(storeLow.GetValues("population"))-1])

	settingsHigh := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pHigh},
				InitStateValues:   append([]float64(nil), initEmpty...),
				Seed:              9,
				StateWidth:        60,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settingsHigh.Init()
	storeHigh := simulator.NewStateTimeStorage()
	implHigh := &simulator.Implementations{
		Iterations:      []simulator.Iteration{&SingleLAPopulationIteration{}},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: storeHigh},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 36,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	configureIterations(settingsHigh, implHigh)
	coordinatorHigh := simulator.NewPartitionCoordinator(settingsHigh, implHigh)
	coordinatorHigh.Run()
	totalHigh := sumState(storeHigh.GetValues("population")[len(storeHigh.GetValues("population"))-1])

	if totalHigh <= totalLow+100 {
		t.Fatalf("expected higher bank-rate path to raise births and stock; low=%f high=%f", totalLow, totalHigh)
	}
}

func TestSingleLA_MeanMonthlyBirthsMatchONSAnnualHull(t *testing.T) {
	path := repoRootDat(t)
	births2019, deaths2019, err := AnnualBirthsAndDeaths(path, "E06000010", 2019)
	if err != nil {
		t.Fatal(err)
	}
	if births2019 != 930 || deaths2019 != 695 {
		t.Fatalf("ONS fixture changed: births=%d deaths=%d", births2019, deaths2019)
	}
	lambda := float64(births2019) / 12.0
	surv, err := LoadSurvivalFracsFromONSJSON(path, "E06000010", 2019)
	if err != nil {
		t.Fatal(err)
	}
	pMap := constCovariateParams(400, 0.5, 15000)
	pMap["survival_fracs"] = surv
	pMap["base_birth_rates"] = []float64{lambda}
	pMap["deterministic"] = []float64{1}
	init := make([]float64, 60)

	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   init,
				Seed:              11,
				StateWidth:        60,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()
	store := simulator.NewStateTimeStorage()
	implBirth := &simulator.Implementations{
		Iterations:      []simulator.Iteration{&SingleLAPopulationIteration{}},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 360,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	configureIterations(settings, implBirth)
	coordinator := simulator.NewPartitionCoordinator(settings, implBirth)
	coordinator.Run()
	vals := store.GetValues("population")
	window := vals[300:]
	if len(window) < 50 {
		t.Fatal("short output window")
	}
	var birthSum float64
	for _, row := range window {
		birthSum += row[0]
	}
	meanB := birthSum / float64(len(window))
	if !scalar.EqualWithinAbs(meanB, lambda, 1e-9) {
		t.Fatalf("mean monthly births %f want %f", meanB, lambda)
	}
}

func TestSingleLA_TwoSectorHarness(t *testing.T) {
	pMap := constCovariateParams(24, 0.25, 8000)
	pMap["sector_hazard_scales"] = []float64{1.0, 1.15}
	pMap["base_birth_rates"] = []float64{5.0, 3.0}
	init := make([]float64, 120)

	settings := &simulator.Settings{
		Iterations: []simulator.IterationSettings{
			{
				Name:              "population",
				Params:            simulator.Params{Map: pMap},
				InitStateValues:   init,
				Seed:              42,
				StateWidth:        120,
				StateHistoryDepth: 2,
			},
		},
		InitTimeValue:         0,
		TimestepsHistoryDepth: 2,
	}
	settings.Init()

	store := simulator.NewStateTimeStorage()
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{&SingleLAPopulationIteration{}},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 24,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	if err := simulator.RunWithHarnesses(settings, impl); err != nil {
		t.Fatal(err)
	}
}

func TestSingleLA_YamlHarness(t *testing.T) {
	settings := simulator.LoadSettingsFromYaml("single_la_population_settings.yaml")
	settings.Init()
	store := simulator.NewStateTimeStorage()
	impl := &simulator.Implementations{
		Iterations:      []simulator.Iteration{&SingleLAPopulationIteration{}},
		OutputCondition: &simulator.EveryStepOutputCondition{},
		OutputFunction:  &simulator.StateTimeStorageOutputFunction{Store: store},
		TerminationCondition: &simulator.NumberOfStepsTerminationCondition{
			MaxNumberOfSteps: 12,
		},
		TimestepFunction: &simulator.ConstantTimestepFunction{Stepsize: 1.0},
	}
	if err := simulator.RunWithHarnesses(settings, impl); err != nil {
		t.Fatal(err)
	}
}