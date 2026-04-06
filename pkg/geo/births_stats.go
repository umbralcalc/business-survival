package geo

import (
	"encoding/json"
	"os"
)

type monthlyBirthTotal struct {
	Month string `json:"month"`
	Total int    `json:"total"`
}

type laBirthBlock struct {
	AreaCode      string              `json:"area_code"`
	MonthlyBirths []monthlyBirthTotal `json:"monthly_births"`
}

type birthsFileMinimal struct {
	Authorities []laBirthBlock `json:"authorities"`
}

// MeanMonthlyTotalBirths returns mean Total over the last lastN months for areaCode
// in la_births-style JSON (cmd/explore output).
func MeanMonthlyTotalBirths(laBirthsJSONPath, areaCode string, lastN int) (float64, error) {
	raw, err := os.ReadFile(laBirthsJSONPath)
	if err != nil {
		return 0, err
	}
	var doc birthsFileMinimal
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, err
	}
	for _, a := range doc.Authorities {
		if a.AreaCode != areaCode {
			continue
		}
		rows := a.MonthlyBirths
		if len(rows) == 0 {
			return 0, nil
		}
		if lastN > len(rows) {
			lastN = len(rows)
		}
		tail := rows[len(rows)-lastN:]
		var sum float64
		for _, r := range tail {
			sum += float64(r.Total)
		}
		return sum / float64(len(tail)), nil
	}
	return 0, nil
}

// DisplacementBirthFactor returns a multiplier in (0,1] shrinking local formation
// when neighbours have high relative birth intensity. leakage in [0,1].
func DisplacementBirthFactor(
	ownCode string,
	birthsPath string,
	lastN int,
	leakage float64,
) float64 {
	if leakage <= 0 {
		return 1.0
	}
	own, err := MeanMonthlyTotalBirths(birthsPath, ownCode, lastN)
	if err != nil || own < 1e-6 {
		return 1.0
	}
	neigh := AdjacentAuthorities[ownCode]
	if len(neigh) == 0 {
		return 1.0
	}
	var nSum float64
	var nc int
	for _, c := range neigh {
		v, err := MeanMonthlyTotalBirths(birthsPath, c, lastN)
		if err != nil || v < 0 {
			continue
		}
		nSum += v
		nc++
	}
	if nc == 0 || nSum < 1e-6 {
		return 1.0
	}
	nAvg := nSum / float64(nc)
	share := nAvg / (own + nAvg + 1e-6)
	f := 1.0 - leakage*share
	if f < 0.5 {
		f = 0.5
	}
	if f > 1.0 {
		f = 1.0
	}
	return f
}
