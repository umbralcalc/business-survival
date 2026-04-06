// Package geo provides postcode → local authority lookups from the ONS
// National Statistics Postcode Lookup (NSPL).
package geo

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// PostcodeLA maps normalised postcodes to local authority district codes (LAUA).
type PostcodeLA map[string]string

// Normalise canonicalises a postcode: uppercase, no spaces.
func Normalise(pc string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(pc), " ", ""))
}

// LoadNSPL reads the NSPL zip archive and returns a postcode → LAUA map.
// It locates the Data/NSPL*.csv file inside the archive automatically.
// If laFilter is non-nil, only postcodes whose LAUA is in the set are kept.
func LoadNSPL(zipPath string, laFilter map[string]bool) (PostcodeLA, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	// Find the main data CSV: Data/NSPL_*.csv (a single large file).
	var dataFile *zip.File
	for _, f := range zr.File {
		name := f.Name
		if strings.HasPrefix(name, "Data/NSPL") && strings.HasSuffix(name, ".csv") {
			dataFile = f
			break
		}
	}
	if dataFile == nil {
		return nil, fmt.Errorf("no Data/NSPL*.csv found in %s", zipPath)
	}

	rc, err := dataFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dataFile.Name, err)
	}
	defer rc.Close()

	r := csv.NewReader(rc)
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	// NSPL renames the LA column periodically (laua → lad25cd etc). Accept any.
	pcdsIdx, lauaIdx, dotermIdx := -1, -1, -1
	for i, col := range header {
		name := strings.ToLower(strings.TrimSpace(col))
		switch {
		case name == "pcds":
			pcdsIdx = i
		case name == "doterm":
			dotermIdx = i
		case name == "laua" || strings.HasPrefix(name, "lad") && strings.HasSuffix(name, "cd"):
			lauaIdx = i
		}
	}
	if pcdsIdx < 0 || lauaIdx < 0 {
		return nil, fmt.Errorf("NSPL header missing pcds or LA column: %v", header)
	}

	out := make(PostcodeLA, 2_500_000)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// Skip terminated postcodes (doterm is set) so we only map live ones.
		if dotermIdx >= 0 && dotermIdx < len(row) && strings.TrimSpace(row[dotermIdx]) != "" {
			continue
		}
		laua := row[lauaIdx]
		if laFilter != nil && !laFilter[laua] {
			continue
		}
		out[Normalise(row[pcdsIdx])] = laua
	}
	return out, nil
}
