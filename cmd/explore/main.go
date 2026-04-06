// explore analyses the Companies House bulk CSV to produce:
//   - Monthly birth rates by sector (from incorporation dates in the live snapshot)
//   - Current stock by sector and incorporation cohort year
//
// Note: this dataset is a snapshot of LIVE companies only. Dissolution dates
// are absent, so survival curves cannot be computed from it. Use the ONS
// Business Demography xlsx (cmd/parse-ons) for survival curves.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Column indices in the Companies House CSV (0-based, after trimming header spaces).
const (
	colPostCode        = 9
	colCompanyCategory = 10
	colCompanyStatus   = 11
	colDissolutionDate = 13
	colIncorporation   = 14
	colSIC1            = 26
)

// SIC group boundaries (first two digits of SIC code).
var sicGroups = map[string]string{
	"41": "Construction",
	"42": "Construction",
	"43": "Construction",
	"47": "Retail",
	"55": "Hospitality",
	"56": "Hospitality",
	"62": "Technology",
	"63": "Technology",
	"69": "Professional",
	"70": "Professional",
	"71": "Professional",
	"72": "Professional",
	"73": "Professional",
	"74": "Professional",
}

func sicGroupFromText(sic string) string {
	sic = strings.TrimSpace(sic)
	if len(sic) < 2 {
		return "Other"
	}
	prefix := sic[:2]
	if g, ok := sicGroups[prefix]; ok {
		return g
	}
	return "Other"
}

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("02/01/2006", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

type cohortKey struct {
	Group string
	Year  int
}

type cohortData struct {
	Births int
}

// MonthlyCount tracks birth and death events by month.
type MonthlyCount struct {
	Month  string `json:"month"` // YYYY-MM
	Births int    `json:"births"`
	Deaths int    `json:"deaths"`
}

// StockByCohort is the surviving count in the live snapshot for one sector and cohort year.
// This represents the numerator of the survival rate if we knew total births.
type StockByCohort struct {
	Sector     string `json:"sector"`
	CohortYear int    `json:"cohort_year"`
	LiveCount  int    `json:"live_count"` // still on register as of snapshot date
}

// Output is the top-level JSON structure.
type Output struct {
	GeneratedAt  string          `json:"generated_at"`
	SnapshotDate string          `json:"snapshot_date"`
	TotalRecords int             `json:"total_records"`
	// MonthlyCounts gives aggregate births (incorporations) and deaths (dissolutions)
	// per month. Deaths will be 0 for this dataset — it covers live companies only.
	MonthlyCounts []MonthlyCount  `json:"monthly_counts"`
	// StockByCohort gives, for each (sector, cohort year), how many companies
	// incorporated in that year are still on the live register at snapshot date.
	// Combine with ONS Business Demography birth counts to derive implied survival rates.
	StockByCohort []StockByCohort `json:"stock_by_cohort"`
}

func main() {
	csvPath := flag.String("csv", "dat/BasicCompanyDataAsOneFile-2026-03-02.csv", "path to Companies House bulk CSV")
	minCohort := flag.Int("min-cohort", 2000, "earliest cohort year to include in survival analysis")
	maxCohort := flag.Int("max-cohort", 2020, "latest cohort year to include (needs >=5 years of follow-up)")
	flag.Parse()

	f, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("open csv: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	// Read header.
	header, err := r.Read()
	if err != nil {
		log.Fatalf("read header: %v", err)
	}
	_ = header

	// Reference date: treat active companies as censored at this date.
cohorts := make(map[cohortKey]*cohortData)
	monthlyBirths := make(map[string]int)
	monthlyDeaths := make(map[string]int)
	total := 0

	log.Printf("Reading CSV...")
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows.
			continue
		}
		if len(row) <= colSIC1 {
			continue
		}

		// Only Private/Public Limited Companies (exclude LLPs, partnerships, etc.)
		cat := strings.TrimSpace(row[colCompanyCategory])
		if cat != "Private Limited Company" && cat != "Public Limited Company" {
			continue
		}

		incorp, ok := parseDate(row[colIncorporation])
		if !ok {
			continue
		}
		total++

		// Monthly birth count.
		monthlyBirths[incorp.Format("2006-01")]++

		// Dissolution date and monthly death count.
		diss, hasDiss := parseDate(row[colDissolutionDate])
		if hasDiss {
			monthlyDeaths[diss.Format("2006-01")]++
		}

		// Survival curve: only for cohorts in range with enough follow-up.
		year := incorp.Year()
		if year < *minCohort || year > *maxCohort {
			continue
		}

		group := sicGroupFromText(row[colSIC1])
		key := cohortKey{Group: group, Year: year}
		if cohorts[key] == nil {
			cohorts[key] = &cohortData{}
		}
		cohorts[key].Births++
	}
	log.Printf("Read %d limited company records", total)

	// Build monthly counts output (births and deaths from 2000 onwards).
	monthSet := make(map[string]bool)
	for m := range monthlyBirths {
		monthSet[m] = true
	}
	for m := range monthlyDeaths {
		monthSet[m] = true
	}
	months := make([]string, 0, len(monthSet))
	for m := range monthSet {
		if m >= "2000-01" {
			months = append(months, m)
		}
	}
	// Sort months.
	for i := 0; i < len(months); i++ {
		for j := i + 1; j < len(months); j++ {
			if months[i] > months[j] {
				months[i], months[j] = months[j], months[i]
			}
		}
	}
	monthlyCounts := make([]MonthlyCount, 0, len(months))
	for _, m := range months {
		monthlyCounts = append(monthlyCounts, MonthlyCount{
			Month:  m,
			Births: monthlyBirths[m],
			Deaths: monthlyDeaths[m],
		})
	}

	// Build stock-by-cohort: count of live companies per (sector, cohort year).
	stock := make([]StockByCohort, 0, len(cohorts))
	for key, cd := range cohorts {
		if cd.Births < 10 {
			continue
		}
		stock = append(stock, StockByCohort{
			Sector:     key.Group,
			CohortYear: key.Year,
			LiveCount:  cd.Births,
		})
	}
	// Sort by sector then cohort year.
	for i := 0; i < len(stock); i++ {
		for j := i + 1; j < len(stock); j++ {
			a, b := stock[i], stock[j]
			if a.Sector > b.Sector || (a.Sector == b.Sector && a.CohortYear > b.CohortYear) {
				stock[i], stock[j] = stock[j], stock[i]
			}
		}
	}

	out := Output{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		SnapshotDate:  "2026-03-02",
		TotalRecords:  total,
		MonthlyCounts: monthlyCounts,
		StockByCohort: stock,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode output: %v", err)
	}
}
