// smcinfer runs packaged stochadex/pkg/analysis SMC calibrations from the CLI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/umbralcalc/business-survival/pkg/calibrate"
	"github.com/umbralcalc/business-survival/pkg/geo"
	"github.com/umbralcalc/business-survival/pkg/population"
)

func main() {
	mode := flag.String("mode", "hazard", "hazard | moments")
	onsPath := flag.String("ons", "dat/ons_demography.json", "ONS JSON")
	birthsPath := flag.String("births", "dat/la_births.json", "explore births JSON")
	la := flag.String("la", "E06000010", "ONS area code")
	cohortYear := flag.Int("cohort-year", 2019, "survival cohort year")
	particles := flag.Int("particles", 96, "SMC particles")
	rounds := flag.Int("rounds", 4, "SMC rounds")
	lookback := flag.Int("birth-lookback", 36, "mean monthly births window")
	verbose := flag.Bool("v", false, "SMC printf diagnostics")
	outPath := flag.String("out", "", "optional JSON path for results")
	flag.Parse()

	surv, err := population.LoadSurvivalFracsFromONSJSON(*onsPath, *la, *cohortYear)
	if err != nil {
		log.Fatal(err)
	}

	switch *mode {
	case "hazard":
		cfg := calibrate.SMCHazardScaleConfig{
			SurvivalFracs:   surv,
			Target5yr:       surv[4],
			LikelihoodSigma: 0.03,
			NParticles:      *particles,
			NRounds:         *rounds,
			PriorLo:         0.3,
			PriorHi:         3.0,
			ProposalSeed:    77,
			Verbose:         *verbose,
		}
		m, s, lm, err := calibrate.RunSMCHazardScaleCalibration(cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("hazard_scale: mean=%.4f std=%.4f log_ml=%.4f\n", m, s, lm)
		writeOut(*outPath, map[string]any{"mode": "hazard", "posterior_mean": m, "posterior_std": s, "log_marginal_lik": lm})

	case "moments":
		mb, err := geo.MeanMonthlyTotalBirths(*birthsPath, *la, *lookback)
		if err != nil {
			log.Fatal(err)
		}
		sigB := math.Max(3.0, mb*0.15)
		cfg := calibrate.SMCPopulationMomentsConfig{
			SurvivalFracs:           surv,
			Target5yr:               surv[4],
			TargetMeanMonthlyBirths: mb,
			BaseBirthRateScalar:     mb,
			Sigma5yr:                0.04,
			SigmaBirths:             sigB,
			NParticles:              *particles,
			NRounds:                 *rounds,
			HazardPriorLo:           0.3,
			HazardPriorHi:           3.0,
			BirthPriorLo:            0.3,
			BirthPriorHi:            3.0,
			ProposalSeed:            88,
			Verbose:                 *verbose,
		}
		hm, hs, bm, bs, lm, err := calibrate.RunSMCPopulationMomentsCalibration(cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("hazard_scale: mean=%.4f std=%.4f\n", hm, hs)
		fmt.Printf("birth_scale:  mean=%.4f std=%.4f\n", bm, bs)
		fmt.Printf("log_ml=%.4f\n", lm)
		writeOut(*outPath, map[string]any{
			"mode": "moments", "hazard_mean": hm, "hazard_std": hs,
			"birth_mean": bm, "birth_std": bs, "log_marginal_lik": lm,
		})

	default:
		log.Fatalf("unknown -mode %q", *mode)
	}
}

func writeOut(path string, doc map[string]any) {
	if path == "" {
		return
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Printf("json: %v", err)
		return
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		log.Printf("write out: %v", err)
	}
}
