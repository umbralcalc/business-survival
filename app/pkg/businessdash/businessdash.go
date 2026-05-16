// Package businessdash is the dexetera dashboard for the
// business-survival post — "Support policies for small business
// survival". The simulator under the hood is a monthly Leslie-style
// model of business birth/age/death dynamics, calibrated to ONS
// business demography and Companies House register data for Kingston
// upon Hull (E06000010), driven by six pre-implemented support
// portfolios under three named macro scenarios.
//
// The controls are two discrete selectors: a portfolio choice (no
// intervention, rates relief, startup grants, incubator/EZ-style,
// mentoring, blended) and a macro scenario choice (baseline,
// recession, expansion). The visualisation has three panels:
//
//   - Portfolio comparison scatter — six dots positioned on (five-year
//     cohort survival %, mean final register stock), one per
//     portfolio, with the user's selection highlighted in the action
//     colour. This is the panel that delivers the multi-objective
//     finding: the "survival winner" sits far left of the "stock
//     winner", and the blended portfolio sits in the top-right
//     trade-off region.
//   - Stock trajectory line chart — live count of total firms on the
//     register over the 10-year horizon, under the user's portfolio +
//     scenario.
//   - Cohort survival decay line chart — live fraction of a 5000-business
//     cohort still surviving as it ages from 0 to 60 months under the
//     same portfolio + scenario.
//
// See app/cmd/business/{register_step,generate} for the wasm entry
// point and the codegen that emits the widget shell respectively.
package businessdash

import (
	"fmt"

	"github.com/umbralcalc/dexetera/pkg/dashboard"
)

// actionColorHex is the magenta the Acting on Simulated Systems
// collection uses to signal "this is what the reader controls". Kept
// in sync with the recolouring constant in cmd/business/generate.
const actionColorHex = "#b0447a"

// referenceColorHex is the slate grey used for static reference
// markers (non-selected portfolios on the scatter, baseline trajectory
// overlays). Matches the AMR / flood / energy widgets' reference hue.
const referenceColorHex = "#7d8aa1"

const (
	// Font sizes are picked so they remain legible after the canvas's
	// usual ~0.6× CSS scale-down in the blog layout.
	titleFontSize   = 22
	axisFontSize    = 18
	captionFontSize = 15
)

// NewConfig returns the dashboard.Config for the business-survival
// widget. Declaration order of renderers matters: later ones draw on
// top. Static frame elements (panel borders, axis lines, tick labels)
// are added first; partition-bound markers (scatter dots, trajectory
// lines, highlights) on top.
func NewConfig() *dashboard.Config {
	vb := dashboard.NewVisualizationBuilder().
		WithCanvas(CanvasWidth, CanvasHeight).
		WithBackground("#fafafa").
		WithUpdateInterval(0)

	// ---- Portfolio comparison scatter (top half) ----

	vb = vb.AddText("", "Portfolio comparison (survival × stock)",
		ScatterX, ScatterY-22,
		&dashboard.TextOptions{
			Color:     "#2c3e50",
			FontSize:  titleFontSize,
			TextAlign: "left",
		})

	// X axis (survival %) — baseline and top.
	vb = vb.AddLine("",
		ScatterX, ScatterY+ScatterHeight,
		ScatterX+ScatterWidth, ScatterY+ScatterHeight,
		&dashboard.LineOptions{Color: "#2c3e50", Width: 1}).
		AddLine("",
			ScatterX, ScatterY,
			ScatterX, ScatterY+ScatterHeight,
			&dashboard.LineOptions{Color: "#2c3e50", Width: 1})

	// X axis tick labels — every 2 pp.
	for s := ScatterMinSurvival; s <= ScatterMaxSurvival+1e-9; s += 0.02 {
		x := survivalToX(s)
		vb = vb.AddLine("",
			int(x), ScatterY+ScatterHeight,
			int(x), ScatterY+ScatterHeight+4,
			&dashboard.LineOptions{Color: "#2c3e50", Width: 1})
		vb = vb.AddText("",
			fmt.Sprintf("%.0f%%", s*100),
			int(x), ScatterY+ScatterHeight+22,
			&dashboard.TextOptions{
				Color:     "#2c3e50",
				FontSize:  axisFontSize,
				TextAlign: "center",
			})
	}

	// Y axis tick labels — every 300 firms.
	for stock := 5400.0; stock <= ScatterMaxStock; stock += 300.0 {
		y := stockToY(stock)
		vb = vb.AddLine("",
			ScatterX-4, int(y),
			ScatterX, int(y),
			&dashboard.LineOptions{Color: "#2c3e50", Width: 1})
		vb = vb.AddText("",
			fmt.Sprintf("%.0f", stock),
			ScatterX-8, int(y)+6,
			&dashboard.TextOptions{
				Color:     "#2c3e50",
				FontSize:  axisFontSize,
				TextAlign: "right",
			})
	}

	// X-axis caption sits below the % tick labels. The Y-axis is
	// captioned implicitly via the title ("survival × stock").
	vb = vb.AddText("",
		"5-yr cohort survival",
		ScatterX+ScatterWidth/2, ScatterY+ScatterHeight+46,
		&dashboard.TextOptions{
			Color:     "#2c3e50",
			FontSize:  captionFontSize,
			TextAlign: "center",
		})

	// Reference dots for all six portfolios (slate grey).
	vb = vb.AddRectangleSet("portfolio_dots", 0, 0, &dashboard.ShapeOptions{
		FillColor: referenceColorHex,
		Anchor:    "topLeft",
	})
	// Highlight marker for the user's current selection.
	vb = vb.AddRectangleSet("portfolio_highlight", 0, 0, &dashboard.ShapeOptions{
		FillColor: actionColorHex,
		Anchor:    "topLeft",
	})

	// Portfolio name labels next to each dot. Positions are
	// hand-picked from the offline Hull reference table so the labels
	// don't crowd each other where dots cluster (rates_relief and
	// blend sit very close on the survival axis; startup and
	// incubator sit very close on the stock axis).
	scenarioForLabels := 0 // baseline scenario for static placement
	labelOffsets := []struct {
		dx, dy int
		align  string
	}{
		{12, -6, "left"},   // baseline (none)
		{-12, -6, "right"}, // rates_relief
		{12, 18, "left"},   // startup
		{12, -6, "left"},   // incubator
		{-12, -6, "right"}, // mentoring
		{12, -6, "left"},   // blend
	}
	for i := 0; i < NumPortfolios; i++ {
		x := survivalToX(ReferenceSurvival[i][scenarioForLabels])
		y := stockToY(ReferenceStock[i][scenarioForLabels])
		off := labelOffsets[i]
		vb = vb.AddText("",
			PortfolioLabels[i],
			int(x)+off.dx, int(y)+off.dy,
			&dashboard.TextOptions{
				Color:     "#2c3e50",
				FontSize:  axisFontSize + 2,
				TextAlign: off.align,
			})
	}

	// ---- Stock trajectory panel (bottom-left) ----

	vb = vb.AddText("", "Stock trajectory",
		StockX, StockY-20,
		&dashboard.TextOptions{
			Color:     "#2c3e50",
			FontSize:  titleFontSize,
			TextAlign: "left",
		})

	vb = vb.AddLine("",
		StockX, StockY+StockHeight,
		StockX+StockWidth, StockY+StockHeight,
		&dashboard.LineOptions{Color: "#2c3e50", Width: 1}).
		AddLine("",
			StockX, StockY,
			StockX, StockY+StockHeight,
			&dashboard.LineOptions{Color: "#2c3e50", Width: 1})

	vb = vb.AddLineChart("stock_trajectory",
		StockX, StockY, StockWidth, StockHeight,
		&dashboard.ChartOptions{
			Color:     "#3c78d8",
			LineWidth: 2,
		})

	vb = vb.AddText("",
		"firms on register",
		StockX, StockY+StockHeight+22,
		&dashboard.TextOptions{
			Color:     "#2c3e50",
			FontSize:  captionFontSize,
			TextAlign: "left",
		})

	// ---- Cohort survival decay panel (bottom-right) ----

	vb = vb.AddText("", "Cohort survival",
		CohortX, CohortY-20,
		&dashboard.TextOptions{
			Color:     "#2c3e50",
			FontSize:  titleFontSize,
			TextAlign: "left",
		})

	vb = vb.AddLine("",
		CohortX, CohortY+CohortHeight,
		CohortX+CohortWidth, CohortY+CohortHeight,
		&dashboard.LineOptions{Color: "#2c3e50", Width: 1}).
		AddLine("",
			CohortX, CohortY,
			CohortX, CohortY+CohortHeight,
			&dashboard.LineOptions{Color: "#2c3e50", Width: 1})

	vb = vb.AddLineChart("cohort_trajectory",
		CohortX, CohortY, CohortWidth, CohortHeight,
		&dashboard.ChartOptions{
			Color:     "#3c78d8",
			LineWidth: 2,
		})

	vb = vb.AddText("",
		"fraction of 5000-firm cohort still active",
		CohortX, CohortY+CohortHeight+22,
		&dashboard.TextOptions{
			Color:     "#2c3e50",
			FontSize:  captionFontSize,
			TextAlign: "left",
		})

	vis := vb.Build()

	cfg := dashboard.NewConfigBuilder("business").
		WithDescription("Small business support policy support: pick a portfolio and macro scenario; the simulator (calibrated to ONS and Companies House data for Kingston upon Hull) shows how five-year cohort survival and register stock shift across the six candidate portfolios over a 10-year horizon. This is a research model fitted to open data, not a policy-design tool.").
		WithServerPartition("population").
		WithServerPartition("cohort").
		WithServerPartition("stock_trajectory").
		WithServerPartition("cohort_trajectory").
		WithServerPartition("portfolio_dots").
		WithServerPartition("portfolio_highlight").
		WithServerPartition("display_progress").
		WithServerPartition("display_outcomes").
		WithActionStatePartition("policy_action").
		WithVisualization(vis).
		WithSimulation(BuildBusinessSimulation)

	// The two top-level discrete sliders are replaced with radio
	// buttons in cmd/business/generate. They're kept in the data model
	// so dexetera's slider→worker action publication mechanism still
	// carries the values to wasm. The labels below are what generate.go
	// uses to find and hide them.
	cfg = cfg.
		WithSlider(dashboard.Slider{
			Name:       "portfolio",
			Label:      "Portfolio (radio-controlled)",
			Partition:  "policy_action",
			ValueIndex: PAIdxPortfolio,
			Min:        0,
			Max:        NumPortfolios - 1,
			Step:       1,
			Default:    PortfolioBaseline,
			Decimals:   0,
		}).
		WithSlider(dashboard.Slider{
			Name:       "scenario",
			Label:      "Scenario (radio-controlled)",
			Partition:  "policy_action",
			ValueIndex: PAIdxScenario,
			Min:        0,
			Max:        NumScenarios - 1,
			Step:       1,
			Default:    ScenarioBaseline,
			Decimals:   0,
		})

	cfg = cfg.
		WithReadout(dashboard.Readout{
			Partition: "display_progress",
			Template:  fmt.Sprintf("month {v%d} of %d · live stock {v%d}", 0, SimMonths, 1),
			Decimals:  0,
		}).
		WithReadout(dashboard.Readout{
			Partition: "display_outcomes",
			Template: fmt.Sprintf(
				"reference: survival {v%d}%% ± {v%d}%% · final stock {v%d} ± {v%d} (64 reps)",
				0, 1, 2, 3,
			),
			Decimals: 1,
		}).
		WithResetButton().
		WithInlineDriver(20)

	return cfg.Build()
}
