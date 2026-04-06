// evalplot renders HTML charts from evaluate JSON (same schema as cmd/evaluate).
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"

	"github.com/umbralcalc/business-survival/pkg/evaluate"
)

func main() {
	inPath := flag.String("in", "dat/evaluate_hull.json", "evaluate JSON output")
	htmlPath := flag.String("html", "dat/evaluate_plot.html", "write interactive HTML (go-echarts, stochadex-style)")
	flag.Parse()

	raw, err := os.ReadFile(*inPath)
	if err != nil {
		log.Fatal(err)
	}

	var outs []evaluate.Output
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		log.Fatal(err)
	}
	if _, ok := probe["items"]; ok {
		var batch evaluate.BatchOutput
		if err := json.Unmarshal(raw, &batch); err != nil {
			log.Fatal(err)
		}
		outs = batch.Items
	} else {
		var one evaluate.Output
		if err := json.Unmarshal(raw, &one); err != nil {
			log.Fatal(err)
		}
		outs = []evaluate.Output{one}
	}
	if len(outs) == 0 {
		log.Fatal("no outputs in JSON")
	}

	page := components.NewPage()
	page.PageTitle = "Policy evaluation"

	for _, out := range outs {
		title := out.AreaName + " (" + out.AreaCode + ")"
		byScenario := map[string]map[string]float64{}
		byScenarioCohort := map[string]map[string]float64{}
		portfolioOrder := map[string]struct{}{}
		var scenarioOrder []string
		seenSc := map[string]struct{}{}
		for _, row := range out.Rows {
			if _, ok := seenSc[row.Scenario]; !ok {
				seenSc[row.Scenario] = struct{}{}
				scenarioOrder = append(scenarioOrder, row.Scenario)
			}
			portfolioOrder[row.PortfolioName] = struct{}{}
		}
		sort.Strings(scenarioOrder)
		var portfolios []string
		for name := range portfolioOrder {
			portfolios = append(portfolios, name)
		}
		sort.Strings(portfolios)

		for _, row := range out.Rows {
			if byScenario[row.Scenario] == nil {
				byScenario[row.Scenario] = make(map[string]float64)
				byScenarioCohort[row.Scenario] = make(map[string]float64)
			}
			byScenario[row.Scenario][row.PortfolioName] = row.MeanFinalStock
			byScenarioCohort[row.Scenario][row.PortfolioName] = row.MeanCohort5yrFrac
		}

		barStock := charts.NewBar()
		barStock.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: title + " — mean final stock"}),
			charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
			charts.WithToolboxOpts(opts.Toolbox{
				Show: opts.Bool(true),
				Feature: &opts.ToolBoxFeature{
					SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{Show: opts.Bool(true)},
				},
			}),
			charts.WithInitializationOpts(opts.Initialization{
				Theme: types.ThemeVintage,
			}),
		)
		barStock.SetXAxis(portfolios)
		for _, sc := range scenarioOrder {
			vals := make([]opts.BarData, len(portfolios))
			for i, p := range portfolios {
				vals[i] = opts.BarData{Value: byScenario[sc][p]}
			}
			barStock.AddSeries(sc, vals)
		}
		page.AddCharts(barStock)

		barSurv := charts.NewBar()
		barSurv.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: title + " — mean 5y cohort survival"}),
			charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
			charts.WithToolboxOpts(opts.Toolbox{
				Show: opts.Bool(true),
				Feature: &opts.ToolBoxFeature{
					SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{Show: opts.Bool(true)},
				},
			}),
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeVintage}),
		)
		barSurv.SetXAxis(portfolios)
		for _, sc := range scenarioOrder {
			vals := make([]opts.BarData, len(portfolios))
			for i, p := range portfolios {
				vals[i] = opts.BarData{Value: byScenarioCohort[sc][p]}
			}
			barSurv.AddSeries(sc, vals)
		}
		page.AddCharts(barSurv)
	}

	f, err := os.Create(*htmlPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := page.Render(f); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", *htmlPath)
}
