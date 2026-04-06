package population

import (
	"encoding/json"
	"fmt"
	"os"
)

// ONSData is the minimal ons_demography.json envelope used by this package.
type ONSData struct {
	SurvivalCurves []SurvivalCurveRecord `json:"survival_curves"`
	Births         []CountRecord         `json:"births"`
	Deaths         []CountRecord         `json:"deaths"`
}

// SurvivalCurveRecord matches one survival_curves[] element.
type SurvivalCurveRecord struct {
	AreaCode    string             `json:"area_code"`
	AreaName    string             `json:"area_name"`
	CohortYear  int                `json:"cohort_year"`
	Births      int                `json:"births"`
	SurvivalPct map[string]float64 `json:"survival_pct"`
}

// CountRecord matches births[] / deaths[] elements.
type CountRecord struct {
	AreaCode string `json:"area_code"`
	AreaName string `json:"area_name"`
	Year     int    `json:"year"`
	Count    int    `json:"count"`
}

// LoadSurvivalFracsFromONSJSON returns cumulative survival fractions
// (years 1–5) for areaCode and cohortYear (e.g. K02000001, 2019 for UK).
func LoadSurvivalFracsFromONSJSON(path, areaCode string, cohortYear int) ([]float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data ONSData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	for _, curve := range data.SurvivalCurves {
		if curve.AreaCode != areaCode || curve.CohortYear != cohortYear {
			continue
		}
		out := make([]float64, 5)
		for y := 1; y <= 5; y++ {
			key := fmt.Sprintf("%d", y)
			pct, ok := curve.SurvivalPct[key]
			if !ok {
				return nil, fmt.Errorf(
					"population: missing survival year %d for %s cohort %d",
					y, areaCode, cohortYear,
				)
			}
			out[y-1] = pct / 100.0
		}
		return out, nil
	}
	return nil, fmt.Errorf(
		"population: no survival curve for area %s cohort %d in %s",
		areaCode, cohortYear, path,
	)
}

// AnnualBirthsAndDeaths returns ONS annual birth and death counts for a LA and year.
// Either value may be zero if absent; err is set only when the file cannot be read
// or unmarshalled.
func AnnualBirthsAndDeaths(path, areaCode string, year int) (births, deaths int, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	var data ONSData
	if err := json.Unmarshal(raw, &data); err != nil {
		return 0, 0, err
	}
	for _, b := range data.Births {
		if b.AreaCode == areaCode && b.Year == year {
			births = b.Count
			break
		}
	}
	for _, d := range data.Deaths {
		if d.AreaCode == areaCode && d.Year == year {
			deaths = d.Count
			break
		}
	}
	return births, deaths, nil
}
