// evaluate runs the single-LA population model under policy portfolios and
// scenarios; see pkg/evaluate for the core engine.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"time"

	"github.com/umbralcalc/business-survival/pkg/evaluate"
	"github.com/umbralcalc/business-survival/pkg/geo"
)

func main() {
	panelPath := flag.String("panel", "dat/la_panel.json", "joined monthly panel JSON")
	birthsPath := flag.String("births", "dat/la_births.json", "per-LA birth JSON from cmd/explore")
	onsPath := flag.String("ons", "dat/ons_demography.json", "ONS survival JSON")
	la := flag.String("la", "E06000010", "ONS / LA code")
	cohortYear := flag.Int("cohort-year", 2019, "ONS survival cohort year")
	runs := flag.Int("runs", 48, "Monte Carlo replications per (portfolio, scenario)")
	stockMonths := flag.Int("months", 120, "months for open-market stock trajectory")
	birthLookback := flag.Int("birth-lookback", 36, "months to average sector birth rates")
	cohortSize := flag.Float64("cohort", 5000, "synthetic cohort size for 5-year survival proxy")
	deterministic := flag.Bool("deterministic", false, "mean-field dynamics (no Poisson/binomial noise)")
	outPath := flag.String("out", "dat/evaluate_output.json", "output path")
	eRate := flag.Float64("e-rate", 0.0, "birth elasticity w.r.t. bank rate (overridden by -auto-elasticities)")
	eClaim := flag.Float64("e-claim", 0.0, "birth elasticity w.r.t. log claimants")
	eDeath := flag.Float64("e-death", 0.0, "death hazard elasticity w.r.t. bank rate")
	eGDP := flag.Float64("e-gdp", 0.06, "birth elasticity w.r.t. GDP growth when -gdp-indexed is set")
	gdpIndexed := flag.Bool("gdp-indexed", false, "synthetic GDP growth series (length = panel months)")
	autoEl := flag.Bool("auto-elasticities", false, "map pooled panel FD regression onto simulation elasticities")
	displacement := flag.Float64("displacement", 0.0, "stylised formation leakage vs geo.AdjacentAuthorities (0–1)")
	distress := flag.Bool("distress-from-claimants", false, "claimant volatility → distress_hazard_boost series")
	bootstrap := flag.Int("bootstrap", 0, "if >0, resample panel months with replacement per replicate")
	batchTarget := flag.Bool("batch-target-las", false, "run all pkg/geo.TargetLAs into one JSON array")
	policyJitter := flag.Float64("policy-jitter", 0, "multiplicative ±noise on policy levers per replicate (non-baseline only)")
	flag.Parse()

	cfg := evaluate.Config{
		PanelPath:             *panelPath,
		BirthsPath:            *birthsPath,
		OnsPath:               *onsPath,
		LACode:                *la,
		CohortYear:            *cohortYear,
		Runs:                  *runs,
		StockMonths:           *stockMonths,
		BirthLookback:         *birthLookback,
		CohortSize:            *cohortSize,
		Deterministic:         *deterministic,
		ERate:                 *eRate,
		EClaim:                *eClaim,
		EDeath:                *eDeath,
		EGDP:                  *eGDP,
		GDPIndexed:            *gdpIndexed,
		AutoElasticities:      *autoEl,
		DisplacementLeak:      *displacement,
		DistressFromClaimants: *distress,
		BootstrapPanels:       *bootstrap,
		PolicyJitter:          *policyJitter,
	}

	if *batchTarget {
		codes := make([]string, 0, len(geo.TargetLAs))
		for c := range geo.TargetLAs {
			codes = append(codes, c)
		}
		sort.Strings(codes)
		batch := evaluate.BatchOutput{}
		for _, code := range codes {
			cfg.LACode = code
			out, err := evaluate.Run(cfg)
			if err != nil {
				log.Printf("skip %s: %v", code, err)
				continue
			}
			batch.Items = append(batch.Items, *out)
		}
		batch.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
		enc, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*outPath, enc, 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote batch %s (%d LAs)", *outPath, len(batch.Items))
		return
	}

	out, err := evaluate.Run(cfg)
	if err != nil {
		log.Fatal(err)
	}
	enc, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*outPath, enc, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%d rows)", *outPath, len(out.Rows))
}
