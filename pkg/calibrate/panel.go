package calibrate

import (
	"encoding/json"
	"math"
	"os"
)

// PanelRow is one monthly observation (matches cmd/analyse la_panel.json).
type PanelRow struct {
	Month         string  `json:"month"`
	Births        int     `json:"births"`
	BankRate      float64 `json:"bank_rate"`
	ClaimantCount int     `json:"claimant_count"`
}

// LAPanel is one authority block from la_panel.json.
type LAPanel struct {
	AreaCode string     `json:"area_code"`
	Rows     []PanelRow `json:"rows"`
}

// PanelFile matches the top-level la_panel.json envelope.
type PanelFile struct {
	Authorities []LAPanel `json:"authorities"`
}

// PooledFirstDiffRegression stacks all LA rows and estimates
//
//	Δ births ~ β_rate Δ bank_rate + β_c Δ log(claimants)
//
// using pooled OLS on first differences.
func PooledFirstDiffRegression(path string) (betaRate, betaLogClaimant float64, n int, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, err
	}
	var doc PanelFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, 0, 0, err
	}
	var sxrr, sxcc, sxrc, xty0, xty1 float64
	for _, la := range doc.Authorities {
		rows := la.Rows
		for i := 1; i < len(rows); i++ {
			db := float64(rows[i].Births - rows[i-1].Births)
			dr := rows[i].BankRate - rows[i-1].BankRate
			c0 := float64(rows[i-1].ClaimantCount)
			c1 := float64(rows[i].ClaimantCount)
			if c0 <= 0 || c1 <= 0 {
				continue
			}
			dlc := math.Log(c1) - math.Log(c0)
			sxrr += dr * dr
			sxcc += dlc * dlc
			sxrc += dr * dlc
			xty0 += dr * db
			xty1 += dlc * db
			n++
		}
	}
	if n < 10 {
		return 0, 0, n, nil
	}
	det := sxrr*sxcc - sxrc*sxrc
	if math.Abs(det) < 1e-18 {
		return 0, 0, n, nil
	}
	betaRate = (xty0*sxcc - xty1*sxrc) / det
	betaLogClaimant = (xty1*sxrr - xty0*sxrc) / det
	return betaRate, betaLogClaimant, n, nil
}

// PanelRecessionWindows returns mean first-differenced births in 2009, the
// broad COVID window (2020-03–2021-02), and all other months (pooled).
func PanelRecessionWindows(path string) (mean2009, meanCOVID, meanRest float64, n09, ncovid, nrest int, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	var doc PanelFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	var sum09, sumCovid, sumRest float64
	for _, la := range doc.Authorities {
		rows := la.Rows
		for i := 1; i < len(rows); i++ {
			m := rows[i].Month
			db := float64(rows[i].Births - rows[i-1].Births)
			switch {
			case m >= "2009-01" && m <= "2009-12":
				sum09 += db
				n09++
			case m >= "2020-03" && m <= "2021-02":
				sumCovid += db
				ncovid++
			default:
				sumRest += db
				nrest++
			}
		}
	}
	if n09 > 0 {
		mean2009 = sum09 / float64(n09)
	}
	if ncovid > 0 {
		meanCOVID = sumCovid / float64(ncovid)
	}
	if nrest > 0 {
		meanRest = sumRest / float64(nrest)
	}
	return mean2009, meanCOVID, meanRest, n09, ncovid, nrest, nil
}
