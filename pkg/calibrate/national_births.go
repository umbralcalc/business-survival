package calibrate

import (
	"encoding/json"
	"os"
)

// LABirthsFile mirrors cmd/explore output (partial).
type LABirthsFile struct {
	Authorities []struct {
		MonthlyBirths []struct {
			Month    string         `json:"month"`
			Total    int            `json:"total"`
			BySector map[string]int `json:"by_sector"`
		} `json:"monthly_births"`
	} `json:"authorities"`
}

// NationalMonthlySectorTotals sums explore output across all authorities.
func NationalMonthlySectorTotals(path string) (map[string]map[string]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc LABirthsFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := make(map[string]map[string]int) // month → sector → count
	for _, la := range doc.Authorities {
		for _, mb := range la.MonthlyBirths {
			if out[mb.Month] == nil {
				out[mb.Month] = make(map[string]int)
			}
			out[mb.Month]["_total"] += mb.Total
			for sec, n := range mb.BySector {
				out[mb.Month][sec] += n
			}
		}
	}
	return out, nil
}

// YearOverYearRatio returns series[month]/series[refMonth] for a sector key
// (_total for all sectors).
func YearOverYearRatio(monthly map[string]map[string]int, month, refMonth, sector string) float64 {
	a := monthly[month]
	b := monthly[refMonth]
	if a == nil || b == nil {
		return 0
	}
	va, vb := a[sector], b[sector]
	if vb == 0 {
		return 0
	}
	return float64(va) / float64(vb)
}
