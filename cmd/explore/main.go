// explore analyses the Companies House bulk CSV and produces, for each
// target local authority:
//
//   - Monthly birth counts (overall + per sector group)
//   - Live-stock counts per sector group per incorporation cohort year
//
// It also emits a national monthly birth aggregate for context.
//
// Postcode → LA resolution uses the ONS NSPL zip in dat/nspl_nov2025.zip.
//
// Note: the Companies House bulk CSV is a snapshot of LIVE companies only;
// dissolution dates are empty. Use cmd/parse-ons for survival curves.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/umbralcalc/business-survival/pkg/geo"
)

// Column indices in the Companies House CSV.
const (
	colPostCode        = 9
	colCompanyCategory = 10
	colIncorporation   = 14
	colSIC1            = 26
)

// SIC group boundaries (first two digits of SIC code).
var sicGroups = map[string]string{
	"41": "Construction", "42": "Construction", "43": "Construction",
	"47": "Retail",
	"55": "Hospitality", "56": "Hospitality",
	"62": "Technology", "63": "Technology",
	"69": "Professional", "70": "Professional", "71": "Professional",
	"72": "Professional", "73": "Professional", "74": "Professional",
}

func sicGroupFromText(sic string) string {
	sic = strings.TrimSpace(sic)
	if len(sic) < 2 {
		return "Other"
	}
	if g, ok := sicGroups[sic[:2]]; ok {
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

// MonthlySectorCount is sector→count for one month.
type MonthlySectorCount struct {
	Month    string         `json:"month"`
	Total    int            `json:"total"`
	BySector map[string]int `json:"by_sector"`
}

// CohortStock is live-register count for one (sector, cohort year).
type CohortStock struct {
	Sector     string `json:"sector"`
	CohortYear int    `json:"cohort_year"`
	LiveCount  int    `json:"live_count"`
}

// LAStats holds everything computed per local authority.
type LAStats struct {
	AreaCode      string               `json:"area_code"`
	AreaName      string               `json:"area_name"`
	TotalLive     int                  `json:"total_live"`
	MonthlyBirths []MonthlySectorCount `json:"monthly_births"`
	StockByCohort []CohortStock        `json:"stock_by_cohort"`
}

// NationalMonthly is the aggregate UK-wide birth count per month.
type NationalMonthly struct {
	Month  string `json:"month"`
	Births int    `json:"births"`
}

type Output struct {
	GeneratedAt    string            `json:"generated_at"`
	SnapshotDate   string            `json:"snapshot_date"`
	NationalTotal  int               `json:"national_total_live"`
	NationalBirths []NationalMonthly `json:"national_monthly_births"`
	Authorities    []LAStats         `json:"authorities"`
}

// internal accumulators
type laAccum struct {
	total   int
	monthly map[string]map[string]int // month → sector → count
	stock   map[string]map[int]int    // sector → cohort year → count
}

func newLAAccum() *laAccum {
	return &laAccum{
		monthly: make(map[string]map[string]int),
		stock:   make(map[string]map[int]int),
	}
}

func main() {
	csvPath := flag.String("csv", "dat/BasicCompanyDataAsOneFile-2026-03-02.csv", "path to Companies House bulk CSV")
	nsplPath := flag.String("nspl", "dat/nspl_nov2025.zip", "path to NSPL postcode zip")
	minCohort := flag.Int("min-cohort", 2000, "earliest cohort year for stock-by-cohort")
	maxCohort := flag.Int("max-cohort", 2025, "latest cohort year for stock-by-cohort")
	flag.Parse()

	log.Printf("Loading NSPL for target LAs...")
	pcLookup, err := geo.LoadNSPL(*nsplPath, geo.TargetLAFilter())
	if err != nil {
		log.Fatalf("load NSPL: %v", err)
	}
	log.Printf("  %d postcodes mapped to %d target LAs", len(pcLookup), len(geo.TargetLAs))

	f, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("open csv: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil {
		log.Fatalf("read header: %v", err)
	}

	accums := make(map[string]*laAccum) // laCode → accumulator
	nationalMonthly := make(map[string]int)
	nationalTotal := 0
	matchedLA := 0

	log.Printf("Reading CSV...")
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(row) <= colSIC1 {
			continue
		}

		cat := strings.TrimSpace(row[colCompanyCategory])
		if cat != "Private Limited Company" && cat != "Public Limited Company" {
			continue
		}
		incorp, ok := parseDate(row[colIncorporation])
		if !ok {
			continue
		}

		nationalTotal++
		month := incorp.Format("2006-01")
		nationalMonthly[month]++

		// Resolve LA via postcode.
		la, ok := pcLookup[geo.Normalise(row[colPostCode])]
		if !ok {
			continue
		}
		matchedLA++

		group := sicGroupFromText(row[colSIC1])
		acc := accums[la]
		if acc == nil {
			acc = newLAAccum()
			accums[la] = acc
		}
		acc.total++

		if acc.monthly[month] == nil {
			acc.monthly[month] = make(map[string]int)
		}
		acc.monthly[month][group]++
		acc.monthly[month]["_total"]++

		year := incorp.Year()
		if year >= *minCohort && year <= *maxCohort {
			if acc.stock[group] == nil {
				acc.stock[group] = make(map[int]int)
			}
			acc.stock[group][year]++
		}
	}
	log.Printf("  %d live limited companies total; %d matched to target LAs", nationalTotal, matchedLA)

	// Build output.
	out := Output{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		SnapshotDate:  "2026-03-02",
		NationalTotal: nationalTotal,
	}

	// National monthly births, sorted.
	natMonths := make([]string, 0, len(nationalMonthly))
	for m := range nationalMonthly {
		if m >= "2000-01" {
			natMonths = append(natMonths, m)
		}
	}
	sort.Strings(natMonths)
	for _, m := range natMonths {
		out.NationalBirths = append(out.NationalBirths, NationalMonthly{
			Month: m, Births: nationalMonthly[m],
		})
	}

	// Per-LA output, ordered by area code for stable diffs.
	laCodes := make([]string, 0, len(accums))
	for code := range accums {
		laCodes = append(laCodes, code)
	}
	sort.Strings(laCodes)

	for _, code := range laCodes {
		acc := accums[code]
		stats := LAStats{
			AreaCode:  code,
			AreaName:  geo.TargetLAs[code],
			TotalLive: acc.total,
		}

		months := make([]string, 0, len(acc.monthly))
		for m := range acc.monthly {
			if m >= "2000-01" {
				months = append(months, m)
			}
		}
		sort.Strings(months)
		for _, m := range months {
			sectors := acc.monthly[m]
			total := sectors["_total"]
			bySector := make(map[string]int, len(sectors)-1)
			for k, v := range sectors {
				if k != "_total" {
					bySector[k] = v
				}
			}
			stats.MonthlyBirths = append(stats.MonthlyBirths, MonthlySectorCount{
				Month: m, Total: total, BySector: bySector,
			})
		}

		// Flatten stock map.
		for sector, years := range acc.stock {
			for year, n := range years {
				stats.StockByCohort = append(stats.StockByCohort, CohortStock{
					Sector: sector, CohortYear: year, LiveCount: n,
				})
			}
		}
		sort.Slice(stats.StockByCohort, func(i, j int) bool {
			a, b := stats.StockByCohort[i], stats.StockByCohort[j]
			if a.Sector != b.Sector {
				return a.Sector < b.Sector
			}
			return a.CohortYear < b.CohortYear
		})

		out.Authorities = append(out.Authorities, stats)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode: %v", err)
	}
}
