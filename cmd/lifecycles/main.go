// lifecycles streams the Companies House bulk CSV and emits a compact JSON
// summary of cross-sectional company ages by sector and target local authority.
//
// Each row is right-censored at the snapshot date for active companies (the
// usual case on the live-companies product). Dissolution dates, when present,
// adjust the age-at-observation for event-time bookkeeping.
//
// Usage:
//
//	go run ./cmd/lifecycles -csv dat/BasicCompanyDataAsOneFile-2026-03-02.csv \
//	    -nspl dat/nspl_nov2025.zip -snapshot 2026-03-02 > dat/lifecycle_age_hist.json
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/umbralcalc/business-survival/pkg/geo"
	"github.com/umbralcalc/business-survival/pkg/lifecycle"
)

type authorityOut struct {
	AreaCode string `json:"area_code"`
	AreaName string `json:"area_name"`
	Hist     map[string][]int `json:"age_by_sector_month_buckets"`
	NRows    int            `json:"n_rows"`
}

type output struct {
	SnapshotAt  string         `json:"snapshot_at"`
	GeneratedAt string         `json:"generated_at"`
	Authorities []authorityOut `json:"authorities"`
}

func main() {
	csvPath := flag.String("csv", "dat/BasicCompanyDataAsOneFile-2026-03-02.csv", "Companies House bulk CSV")
	nsplPath := flag.String("nspl", "dat/nspl_nov2025.zip", "ONS NSPL postcode zip")
	snapshotStr := flag.String("snapshot", "2026-03-02", "snapshot date YYYY-MM-DD")
	flag.Parse()

	snapshot, err := time.Parse("2006-01-02", *snapshotStr)
	if err != nil {
		log.Fatalf("snapshot: %v", err)
	}

	pcLookup, err := geo.LoadNSPL(*nsplPath, geo.TargetLAFilter())
	if err != nil {
		log.Fatalf("load NSPL: %v", err)
	}

	acc := make(map[string]*lifecycle.SectorAgeHistogram)
	for code := range geo.TargetLAs {
		acc[code] = lifecycle.NewSectorAgeHistogram()
	}

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
		log.Fatalf("header: %v", err)
	}

	var nScan, nMatch int
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		nScan++
		la, ok := pcLookup[geo.Normalise(row[lifecycle.ColPostCode])]
		if !ok {
			continue
		}
		rec, ok := lifecycle.ParseRow(row, la, snapshot)
		if !ok {
			continue
		}
		acc[la].Add(rec)
		nMatch++
	}
	log.Printf("scanned %d rows; matched %d to target LAs", nScan, nMatch)

	keys := make([]string, 0, len(acc))
	for k := range acc {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var auth []authorityOut
	for _, code := range keys {
		h := acc[code]
		if h.NRows == 0 {
			continue
		}
		bySec := make(map[string][]int)
		for sec, arr := range h.BySector {
			sl := make([]int, lifecycle.AgeBucketCount)
			copy(sl, arr[:])
			bySec[sec] = sl
		}
		auth = append(auth, authorityOut{
			AreaCode: code,
			AreaName: geo.TargetLAs[code],
			Hist:     bySec,
			NRows:    h.NRows,
		})
	}

	out := output{
		SnapshotAt:  snapshot.Format("2006-01-02"),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Authorities: auth,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode: %v", err)
	}
}
