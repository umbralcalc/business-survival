// analyse joins the per-LA monthly birth series produced by cmd/explore with
// macroeconomic covariates (BoE Bank Rate and NOMIS claimant count) and prints
// a summary of each LA's time-series statistics and Pearson correlations.
//
// Usage:
//
//	go run ./cmd/analyse \
//	    -births dat/la_births.json \
//	    -boe dat/boe_bank_rate.csv \
//	    -claimant dat/nomis_claimant_count_la_2013_2026.csv \
//	    > dat/la_panel.json
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/umbralcalc/business-survival/pkg/geo"
)

// LABirthsInput mirrors the output schema of cmd/explore.
type LABirthsInput struct {
	Authorities []struct {
		AreaCode      string `json:"area_code"`
		AreaName      string `json:"area_name"`
		TotalLive     int    `json:"total_live"`
		MonthlyBirths []struct {
			Month    string         `json:"month"`
			Total    int            `json:"total"`
			BySector map[string]int `json:"by_sector"`
		} `json:"monthly_births"`
	} `json:"authorities"`
}

// MonthlyRow is one joined observation for one LA.
type MonthlyRow struct {
	Month         string  `json:"month"`
	Births        int     `json:"births"`
	BankRate      float64 `json:"bank_rate"`       // end-of-month Bank Rate (%)
	ClaimantCount int     `json:"claimant_count"`  // NOMIS claimant count
}

// LAPanel is the joined time series + correlations for one LA.
type LAPanel struct {
	AreaCode               string       `json:"area_code"`
	AreaName               string       `json:"area_name"`
	TotalLive              int          `json:"total_live"`
	NObservations          int          `json:"n_observations"`
	MeanBirths             float64      `json:"mean_births"`
	StdBirths              float64      `json:"std_births"`
	CorrBirthsBankRate     float64      `json:"corr_births_bank_rate"`
	CorrBirthsClaimant     float64      `json:"corr_births_claimant"`
	CorrBirthsBankRateLag3 float64      `json:"corr_births_bank_rate_lag3"` // births vs rate 3 months earlier
	Rows                   []MonthlyRow `json:"rows"`
}

type Output struct {
	GeneratedAt string    `json:"generated_at"`
	WindowFrom  string    `json:"window_from"`
	WindowTo    string    `json:"window_to"`
	Authorities []LAPanel `json:"authorities"`
}

// loadBoE reads dat/boe_bank_rate.csv and returns month → end-of-month rate.
// The series is daily so we take the last observation of each month.
func loadBoE(path string) (map[string]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil {
		return nil, err
	}
	// Temporary: month → (maxDay, rate).
	type dr struct {
		day  int
		rate float64
	}
	byMonth := make(map[string]dr)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) < 2 {
			continue
		}
		t, err := time.Parse("02 Jan 2006", row[0])
		if err != nil {
			continue
		}
		rate, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			continue
		}
		m := t.Format("2006-01")
		cur, ok := byMonth[m]
		if !ok || t.Day() > cur.day {
			byMonth[m] = dr{day: t.Day(), rate: rate}
		}
	}
	out := make(map[string]float64, len(byMonth))
	for m, v := range byMonth {
		out[m] = v.rate
	}
	return out, nil
}

// loadClaimant reads the NOMIS claimant count CSV and returns
// (laCode, month) → count.
func loadClaimant(path string) (map[[2]string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	dateIdx, codeIdx, valIdx := -1, -1, -1
	for i, c := range header {
		switch strings.ToUpper(strings.Trim(c, `"`)) {
		case "DATE_NAME":
			dateIdx = i
		case "GEOGRAPHY_CODE":
			codeIdx = i
		case "OBS_VALUE":
			valIdx = i
		}
	}
	if dateIdx < 0 || codeIdx < 0 || valIdx < 0 {
		return nil, fmt.Errorf("claimant CSV missing required columns: %v", header)
	}

	out := make(map[[2]string]int)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// NOMIS DATE_NAME is e.g. "January 2013". Parse to YYYY-MM.
		t, err := time.Parse("January 2006", strings.TrimSpace(row[dateIdx]))
		if err != nil {
			continue
		}
		month := t.Format("2006-01")
		code := strings.TrimSpace(row[codeIdx])
		n, _ := strconv.Atoi(strings.TrimSpace(row[valIdx]))
		out[[2]string{code, month}] = n
	}
	return out, nil
}

// pearson computes the Pearson correlation between two equal-length slices.
// Returns NaN if either has zero variance or length < 2.
func pearson(x, y []float64) float64 {
	n := len(x)
	if n < 2 || n != len(y) {
		return math.NaN()
	}
	var sx, sy float64
	for i := 0; i < n; i++ {
		sx += x[i]
		sy += y[i]
	}
	mx := sx / float64(n)
	my := sy / float64(n)
	var num, dx2, dy2 float64
	for i := 0; i < n; i++ {
		dx := x[i] - mx
		dy := y[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}
	if dx2 == 0 || dy2 == 0 {
		return math.NaN()
	}
	return num / math.Sqrt(dx2*dy2)
}

// round rounds to dp decimal places. NaN/Inf are coerced to 0 so the result
// is always JSON-encodable; callers lose NaN signal but this is acceptable
// for our summary statistics.
func round(x float64, dp int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	p := math.Pow(10, float64(dp))
	return math.Round(x*p) / p
}

func main() {
	birthsPath := flag.String("births", "dat/la_births.json", "path to cmd/explore output")
	boePath := flag.String("boe", "dat/boe_bank_rate.csv", "path to BoE Bank Rate CSV")
	claimantPath := flag.String("claimant", "dat/nomis_claimant_count_la_2013_2026.csv", "path to NOMIS claimant count CSV")
	windowFrom := flag.String("from", "2013-01", "start month (inclusive) for the joined panel")
	windowTo := flag.String("to", "2025-12", "end month (inclusive)")
	flag.Parse()

	log.Printf("Loading births: %s", *birthsPath)
	birthsFile, err := os.Open(*birthsPath)
	if err != nil {
		log.Fatalf("open births: %v", err)
	}
	var births LABirthsInput
	if err := json.NewDecoder(birthsFile).Decode(&births); err != nil {
		log.Fatalf("decode births: %v", err)
	}
	birthsFile.Close()

	log.Printf("Loading BoE Bank Rate: %s", *boePath)
	boe, err := loadBoE(*boePath)
	if err != nil {
		log.Fatalf("load BoE: %v", err)
	}

	log.Printf("Loading NOMIS claimant count: %s", *claimantPath)
	claimant, err := loadClaimant(*claimantPath)
	if err != nil {
		log.Fatalf("load claimant: %v", err)
	}

	out := Output{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		WindowFrom:  *windowFrom,
		WindowTo:    *windowTo,
	}

	for _, la := range births.Authorities {
		// Index birth months for quick lookup.
		bm := make(map[string]int)
		for _, m := range la.MonthlyBirths {
			bm[m.Month] = m.Total
		}
		// Walk the window month by month.
		months := monthsInRange(*windowFrom, *windowTo)
		var rows []MonthlyRow
		var b, r, c []float64
		for _, m := range months {
			births := bm[m]
			rate, hasRate := boe[m]
			if !hasRate {
				continue
			}
			// Try current code then any legacy aliases (e.g. Sheffield E08000039→E08000019).
			var count int
			hasCount := false
			for _, code := range geo.AllCodesForLA(la.AreaCode) {
				if v, ok := claimant[[2]string{code, m}]; ok {
					count = v
					hasCount = true
					break
				}
			}
			if !hasCount {
				continue
			}
			rows = append(rows, MonthlyRow{
				Month: m, Births: births, BankRate: rate, ClaimantCount: count,
			})
			b = append(b, float64(births))
			r = append(r, rate)
			c = append(c, float64(count))
		}

		mean, std := meanStd(b)
		// Lag-3 correlation: births[t] vs rate[t-3].
		var lagCorr float64
		if len(b) > 3 {
			lagCorr = pearson(b[3:], r[:len(r)-3])
		} else {
			lagCorr = math.NaN()
		}

		out.Authorities = append(out.Authorities, LAPanel{
			AreaCode:               la.AreaCode,
			AreaName:               la.AreaName,
			TotalLive:              la.TotalLive,
			NObservations:          len(rows),
			MeanBirths:             round(mean, 2),
			StdBirths:              round(std, 2),
			CorrBirthsBankRate:     round(pearson(b, r), 4),
			CorrBirthsClaimant:     round(pearson(b, c), 4),
			CorrBirthsBankRateLag3: round(lagCorr, 4),
			Rows:                   rows,
		})
	}

	// Sort authorities by code for stable output.
	sort.Slice(out.Authorities, func(i, j int) bool {
		return out.Authorities[i].AreaCode < out.Authorities[j].AreaCode
	})

	// Pretty summary to stderr.
	fmt.Fprintf(os.Stderr, "\n%-30s %6s %8s %8s %8s %12s\n",
		"LA", "n", "mean_b", "std_b", "ρ(rate)", "ρ(claimant)")
	for _, p := range out.Authorities {
		fmt.Fprintf(os.Stderr, "%-30s %6d %8.1f %8.1f %8.3f %12.3f\n",
			p.AreaName, p.NObservations, p.MeanBirths, p.StdBirths,
			p.CorrBirthsBankRate, p.CorrBirthsClaimant)
	}

	// Also suppress Rows in summary to keep output compact if requested?
	// For now emit the full panel.
	_ = geo.TargetLAs // keep import for future use

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode: %v", err)
	}
}

func monthsInRange(from, to string) []string {
	start, err := time.Parse("2006-01", from)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01", to)
	if err != nil {
		return nil
	}
	var out []string
	for t := start; !t.After(end); t = t.AddDate(0, 1, 0) {
		out = append(out, t.Format("2006-01"))
	}
	return out
}

func meanStd(xs []float64) (float64, float64) {
	n := len(xs)
	if n == 0 {
		return 0, 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	m := s / float64(n)
	if n < 2 {
		return m, 0
	}
	var v float64
	for _, x := range xs {
		d := x - m
		v += d * d
	}
	return m, math.Sqrt(v / float64(n-1))
}
