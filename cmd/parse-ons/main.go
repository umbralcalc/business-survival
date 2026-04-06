// parse-ons extracts survival curves and birth/death counts from the ONS
// Business Demography reference table Excel file and writes JSON to stdout.
//
// Usage:
//
//	go run ./cmd/parse-ons -xlsx dat/ons_business_demography_2024.xlsx \
//	    > dat/ons_demography.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Sheet → cohort/year. Structure matches the 2024 publication.
var (
	survivalSheets = map[string]int{
		"Table 5.1a": 2019,
		"Table 5.1b": 2020,
		"Table 5.1c": 2021,
		"Table 5.1d": 2022,
		"Table 5.1e": 2023,
	}
	birthSheets = map[string]int{
		"Table 1.1a": 2019,
		"Table 1.1b": 2020,
		"Table 1.1c": 2021,
		"Table 1.1d": 2022,
	}
	deathSheets = map[string]int{
		"Table 2.1a": 2019,
		"Table 2.1b": 2020,
		"Table 2.1c": 2021,
		"Table 2.1d": 2022,
	}
)

type SurvivalCurve struct {
	AreaCode    string          `json:"area_code"`
	AreaName    string          `json:"area_name"`
	CohortYear  int             `json:"cohort_year"`
	Births      int             `json:"births"`
	SurvivalPct map[int]float64 `json:"survival_pct"` // year → percent
}

type Count struct {
	AreaCode string `json:"area_code"`
	AreaName string `json:"area_name"`
	Year     int    `json:"year"`
	Count    int    `json:"count"`
}

type Output struct {
	GeneratedAt    string          `json:"generated_at"`
	Source         string          `json:"source"`
	SurvivalCurves []SurvivalCurve `json:"survival_curves"`
	Births         []Count         `json:"births"`
	Deaths         []Count         `json:"deaths"`
}

// dataRows reads a sheet and yields rows skipping the 4 title/header rows.
// It returns [][]string where each inner slice is one data row's cells.
func dataRows(f *excelize.File, sheet string) ([][]string, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) <= 4 {
		return nil, nil
	}
	return rows[4:], nil
}

// parseInt accepts ONS counts which may be blank, rounded integers, or have
// trailing decimals. Returns ok=false for blank.
// Excelize's GetRows returns formatted display values, so integers may be
// comma-separated ("363,825") and floats may appear as "94.6".
func cleanNumeric(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), ",", "")
}

func parseInt(s string) (int, bool) {
	s = cleanNumeric(s)
	if s == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, true
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f), true
	}
	return 0, false
}

func parseFloat(s string) (float64, bool) {
	s = cleanNumeric(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func cell(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

func parseSurvival(f *excelize.File) ([]SurvivalCurve, error) {
	var out []SurvivalCurve
	sheets := make([]string, 0, len(survivalSheets))
	for s := range survivalSheets {
		sheets = append(sheets, s)
	}
	sort.Strings(sheets)

	for _, sheet := range sheets {
		cohort := survivalSheets[sheet]
		rows, err := dataRows(f, sheet)
		if err != nil {
			// Missing sheet is tolerated (publication may differ).
			log.Printf("skip %s: %v", sheet, err)
			continue
		}
		for _, row := range rows {
			code := strings.TrimSpace(cell(row, 0))
			if code == "" {
				continue
			}
			name := strings.TrimSpace(cell(row, 1))
			births, ok := parseInt(cell(row, 2))
			if !ok || births == 0 {
				continue
			}
			// Columns 3,4 = 1yr count,pct ; 5,6 = 2yr ; ... up to 5yr.
			pct := make(map[int]float64)
			for yr := 1; yr <= 5; yr++ {
				pctIdx := 2 + yr*2 // 4, 6, 8, 10, 12
				if v, ok := parseFloat(cell(row, pctIdx)); ok {
					pct[yr] = math.Round(v*100) / 100
				}
			}
			if len(pct) == 0 {
				continue
			}
			out = append(out, SurvivalCurve{
				AreaCode:    code,
				AreaName:    name,
				CohortYear:  cohort,
				Births:      births,
				SurvivalPct: pct,
			})
		}
	}
	return out, nil
}

func parseCounts(f *excelize.File, sheets map[string]int) ([]Count, error) {
	var out []Count
	names := make([]string, 0, len(sheets))
	for s := range sheets {
		names = append(names, s)
	}
	sort.Strings(names)

	for _, sheet := range names {
		year := sheets[sheet]
		rows, err := dataRows(f, sheet)
		if err != nil {
			log.Printf("skip %s: %v", sheet, err)
			continue
		}
		for _, row := range rows {
			code := strings.TrimSpace(cell(row, 0))
			if code == "" {
				continue
			}
			name := strings.TrimSpace(cell(row, 1))
			count, ok := parseInt(cell(row, 2))
			if !ok {
				continue
			}
			out = append(out, Count{
				AreaCode: code,
				AreaName: name,
				Year:     year,
				Count:    count,
			})
		}
	}
	return out, nil
}

func main() {
	xlsxPath := flag.String("xlsx", "dat/ons_business_demography_2024.xlsx", "path to ONS xlsx")
	flag.Parse()

	f, err := excelize.OpenFile(*xlsxPath)
	if err != nil {
		log.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	curves, err := parseSurvival(f)
	if err != nil {
		log.Fatalf("parse survival: %v", err)
	}
	births, err := parseCounts(f, birthSheets)
	if err != nil {
		log.Fatalf("parse births: %v", err)
	}
	deaths, err := parseCounts(f, deathSheets)
	if err != nil {
		log.Fatalf("parse deaths: %v", err)
	}

	out := Output{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		Source:         "ONS Business Demography 2024",
		SurvivalCurves: curves,
		Births:         births,
		Deaths:         deaths,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode: %v", err)
	}
	fmt.Fprintf(os.Stderr,
		"Extracted: %d survival series, %d birth rows, %d death rows\n",
		len(curves), len(births), len(deaths))
}
