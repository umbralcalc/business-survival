package policy

// ScenarioLabel names macro paths applied on top of observed panel covariates.
type ScenarioLabel string

const (
	ScenarioBaseline   ScenarioLabel = "baseline"
	ScenarioRecession  ScenarioLabel = "recession"
	ScenarioExpansion  ScenarioLabel = "expansion"
)

// AllScenarioLabels is the default evaluation set.
var AllScenarioLabels = []ScenarioLabel{
	ScenarioBaseline,
	ScenarioRecession,
	ScenarioExpansion,
}

// AdjustCovariates returns copies of bank-rate and claimant series with macro
// overlays. GDP growth series is optional (nil = omit from caller's param map).
func AdjustCovariates(
	rates []float64,
	claimants []float64,
	gdp []float64,
	sc ScenarioLabel,
) (ratesOut, claimantsOut, gdpOut []float64) {
	ratesOut = append([]float64(nil), rates...)
	claimantsOut = append([]float64(nil), claimants...)
	if len(gdp) > 0 {
		gdpOut = append([]float64(nil), gdp...)
	}
	switch sc {
	case ScenarioBaseline:
		return ratesOut, claimantsOut, gdpOut
	case ScenarioRecession:
		for i := range ratesOut {
			ratesOut[i] += 0.02 // +200 bps stress vs observed path
		}
		for i := range claimantsOut {
			claimantsOut[i] *= 1.15
		}
		for i := range gdpOut {
			gdpOut[i] -= 2.0 // percentage points vs baseline growth path
		}
	case ScenarioExpansion:
		for i := range ratesOut {
			ratesOut[i] -= 0.01
		}
		for i := range claimantsOut {
			claimantsOut[i] *= 0.92
		}
		for i := range gdpOut {
			gdpOut[i] += 1.5
		}
	default:
		return ratesOut, claimantsOut, gdpOut
	}
	return ratesOut, claimantsOut, gdpOut
}
