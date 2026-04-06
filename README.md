# Business survival & support policy simulation — project report

**Stack:** [stochadex](https://github.com/umbralcalc/stochadex) · Go · ONS business demography · Companies House bulk data · NOMIS / Bank of England panels

This repository is a **stochastic simulation and decision layer** for local business demography: birth–age–death dynamics calibrated to open UK data, with **literature-informed support portfolios** compared under **macro scenarios**. This document is the **project report** for collaborators and readers reinstantiating the work.

---

## Executive summary

We connect **ONS survival curves**, **per-LA monthly formation** (from the Companies House live register and NSPL postcode geography), and **economic covariates** (Bank Rate, claimant count) into a **monthly Leslie-style model** (`SingleLAPopulationIteration`) with optional **Sequential Monte Carlo (SMC)** calibration via stochadex **`pkg/analysis`**. A **policy layer** encodes several intervention bundles as multipliers on births and hazards; **`cmd/evaluate`** runs Monte Carlo over portfolios × scenarios and writes JSON; **`cmd/evalplot`** turns that into interactive HTML charts.

**Illustrative result (Kingston upon Hull, `E06000010`):** under a 120‑month run with 64 replications, a **rates & cash-flow relief** bundle and a **blended** bundle raise the model’s **five-year cohort survival** by about **four percentage points** versus baseline (baseline ~37.4% vs relief ~41.4% in the baseline macro path; see §5). **Startup-style** tilts raise **stock** more than **pure survival**, as encoded. These numbers are **ranking and stress-test instruments**, not departmental forecasts (see caveats).

**How far that goes toward the original research question:** we can now answer the **portfolio ranking** part for **modelled survival and register stock** in one LA and three stylised macro paths; we **cannot** yet answer **employment**, **cost–benefit**, **displacement magnitude**, or **“which sectors win”** as separate reportable outcomes—§1 and §5.4 spell that out, and §6 lists the extensions needed.

---

## 1. Purpose and research question

Small-business demography is stochastic and policy-relevant—rates relief, grants, zones, mentoring—but most evidence is **before/after or regression**, not **forward simulation under interventions**.

The **full question** this project was set up to support:

> *Given **current economic conditions** and the **business population in a region**, which **combination of support interventions** (rate relief, grants, incubators, mentoring, **etc.**) yields the **greatest improvement in survival and employment-related outcomes over 5–10 years**, **for which sectors and business types**, and **how robust** is that under **recession vs expansion**, **displacement**, and **value for money**?*

The implementation deliberately uses **freely available** UK statistics and bulk registration data so the pipeline can be **replayed and audited**. The **narrower operational question** the running code answers **today** is:

> *For a chosen LA, which **encoded portfolio** improves **modelled 5‑year cohort survival** and/or **mean simulated stock** most, and does that **ranking** hold under **baseline / recession / expansion** covariate paths?*

Everything else in the **full** question is **future work** (see §6) unless noted below.

---

## 2. What we built

| Piece | Role |
|--------|------|
| `pkg/population` | Monthly multi-sector Leslie; ONS-linked hazards; economic & policy multipliers; `RunToState` harness; SMC helper iterations (`ScaledCohortSurvivalIteration`, `PopulationSurvivalBirthMomentsIteration`). |
| `pkg/calibrate` | Panel FD regression, COVID/national birth patterns, hazard scaling, **SMC** (`RunSMCHazardScaleCalibration`, `RunSMCPopulationMomentsCalibration`), panel→elasticity mapping. |
| `pkg/lifecycle` | CH row parsing, SIC→sector, age histograms. |
| `pkg/policy` | Portfolios, literature priors table, scenarios (baseline / recession / expansion). |
| `pkg/geo` | Target LAs, adjacency sketch for displacement, birth-rate helpers. |
| `pkg/evaluate` | `Run(Config)` — Monte Carlo evaluation engine. |
| `cmd/parse-ons`, `cmd/explore`, `cmd/analyse` | Build `dat/ons_demography.json`, `dat/la_births.json`, `dat/la_panel.json`. |
| `cmd/lifecycles` | CH → JSON age histograms by LA/sector. |
| `cmd/evaluate`, `cmd/evalplot`, `cmd/smcinfer` | Policy runs, charts, SMC CLI. |
| `CLAUDE.md` | Contributor notes for stochadex iteration patterns and YAML. |

---

## 3. Methods (concise)

- **Demography:** Businesses live in **sector × age (month) buckets** (60 months + top bucket). Hazards derive from **ONS cumulative survival** (years 1–5); sector **relative** hazards follow a literature-style table, then a **global scale** matches five-year survival for the LA mix.
- **Covariates:** Bank Rate and claimant count (panel) enter **log‑linear** birth and hazard scaling; optional GDP path; optional **distress** series (claimant volatility) scales hazard.
- **Policies:** Portfolios map to `policy_*` parameters (global and per‑sector birth/hazard/“infant” hazard). **Displacement** optionally scales formation using neighbour mean births (`geo.AdjacentAuthorities`).
- **Scenarios:** Observed rate/claimant paths are overlaid with stylised **recession** (tighter rates, higher claimants) or **expansion**.
- **Inference:** **Moment matching** + pooled FD regression; **SMC** through **`analysis.RunSMCInference`** with **`inference.DataComparisonIteration`** (scalar hazard or bivariate survival+birth moments).

---

## 4. How to use

### 4.1 Prerequisites

- **Go** (see `go.mod`).
- Data under `dat/` — either your own downloads or outputs from the commands below (large CH CSV and NSPL zip are not committed).

### 4.2 Build and test

```bash
go build ./...
go test -count=1 ./...
```

### 4.3 Run the stochadex example (YAML-generated partition)

```bash
go run github.com/umbralcalc/stochadex/cmd/stochadex --config cfg/single_la_population.yaml
```

### 4.4 Rebuild analysis artefacts (paths are examples)

```bash
# ONS survival / births / deaths JSON → dat/ons_demography.json
go run ./cmd/parse-ons -h

# Companies House bulk CSV + NSPL → dat/la_births.json (see cmd/explore flags)
go run ./cmd/explore -csv dat/BasicCompanyDataAsOneFile-2026-03-02.csv \
  -nspl dat/nspl_nov2025.zip

# Panel: births + BoE rate + claimants → dat/la_panel.json
go run ./cmd/analyse -h

# Cross-sectional age by sector / LA from CH CSV
go run ./cmd/lifecycles -csv dat/BasicCompanyDataAsOneFile-2026-03-02.csv \
  -nspl dat/nspl_nov2025.zip -snapshot 2026-03-02 > dat/lifecycle_age_hist.json
```

### 4.5 Policy evaluation (single LA)

```bash
go run ./cmd/evaluate -la E06000010 -runs 64 -months 120 -out dat/evaluate_hull.json
```

**Useful flags:** `-auto-elasticities` · `-displacement 0.12` · `-distress-from-claimants` · `-bootstrap 30` · `-policy-jitter 0.08` · `-deterministic` · `-gdp-indexed`

### 4.6 Batch target local authorities

```bash
go run ./cmd/evaluate -batch-target-las -runs 32 -months 96 -auto-elasticities \
  -out dat/evaluate_batch.json
```

### 4.7 Charts from evaluation JSON

Uses [go-echarts](https://github.com/go-echarts/go-echarts) (same ecosystem as stochadex tests):

```bash
go run ./cmd/evalplot -in dat/evaluate_hull.json -html dat/evaluate_hull.html
```

Open the HTML in a browser.

### 4.8 SMC calibration (stochadex `pkg/analysis`)

```bash
# Scalar hazard multiplier vs 5‑year survival
go run ./cmd/smcinfer -mode hazard

# Bivariate: survival + mean monthly births
go run ./cmd/smcinfer -mode moments -la E06000010 -particles 80 -rounds 4 -out dat/smc_moments.json
```

---

## 5. Results

### 5.1 National context (ONS)

From the Phase‑1 exploratory pass: the **2019 birth cohort** at **UK** level has a **five‑year survival** of about **38.4%** (ONS business demography, as cited in project notes). That figure is a **benchmark** for qualitative comparison only—the simulation is LA‑specific and uses a structural model, not a reproduction of the national estimator.

### 5.2 Case study: Kingston upon Hull (`E06000010`)

The following comes from **`dat/evaluate_hull.json`** (`generated_at`: **2026-04-06T09:15:15Z**): **64** stochastic replications per cell, **120** months of stock dynamics, cohort survival sub‑run with **births switched off** and cohort size **5000** (default). **Macro scenarios:** `baseline`, `recession`, `expansion` (see `pkg/policy/scenarios.go`).

**Five-year cohort survival** (mean ± std dev of replicate means):

| Portfolio | Baseline | Recession | Expansion |
|-----------|----------|-----------|------------|
| No additional intervention | 0.374 ± 0.007 | 0.372 ± 0.007 | 0.375 ± 0.007 |
| Rates & cash-flow relief | **0.414** ± 0.007 | **0.412** ± 0.007 | **0.414** ± 0.008 |
| Startup finance & first-year support | 0.374 ± 0.007 | 0.372 ± 0.007 | 0.374 ± 0.007 |
| Incubator / enterprise-zone style | 0.385 ± 0.006 | 0.384 ± 0.007 | 0.386 ± 0.007 |
| Mentoring & peer resilience | 0.398 ± 0.006 | 0.394 ± 0.007 | 0.398 ± 0.008 |
| Blended portfolio | **0.413** ± 0.007 | **0.411** ± 0.007 | **0.415** ± 0.006 |

**Mean final stock** (same run; arbitrary scale driven by Poisson births and calibration—compare **across portfolios**, not to official stock counts):

| Portfolio | Baseline | Recession | Expansion |
|-----------|----------|-----------|------------|
| Baseline | ~5625 ± 72 | ~5557 ± 78 | ~5662 ± 81 |
| Rates relief | ~6042 ± 85 | ~5953 ± 72 | ~6050 ± 77 |
| Startup grants | ~6280 ± 76 | ~6195 ± 82 | ~6294 ± 77 |
| Incubator | ~6225 ± 78 | ~6151 ± 77 | ~6256 ± 74 |
| Mentoring | ~5963 ± 80 | ~5911 ± 59 | ~6010 ± 71 |
| Blended | **~6485** ± 82 | **~6401** ± 72 | **~6534** ± 80 |

**Reading:** In this calibration, **relief** and **blended** packages lift **cohort survival** most; **startup** tilts mainly lift **flows/stock** without moving the pure survival metric much, consistent with how those levers are parameterised. **Recession** shaves outcomes slightly relative to **baseline**; rankings are **stable** across the three macro overlays in this run.

### 5.3 What we found out — mapped to the original question

The table below ties **empirical results** (Hull case, `dat/evaluate_hull.json`) to the **full** research question. Where a cell says *not in model*, the tool does not yet produce that outcome.

| Theme | Original question asks… | What we found **so far** |
|--------|-------------------------|---------------------------|
| **Which portfolio “wins”?** | Best **combination** of relief, grants, incubators, mentoring | **Survival-oriented objective:** **rates & cash-flow relief** and the **blended** bundle rank highest for **5‑year cohort survival** (~**+4 pp** vs no intervention in the baseline scenario). **Stock-oriented objective:** **startup / first‑year** and **blended** rank highest for **mean final simulated stock**. **Incubator/EZ-style** is intermediate on both. **Mentoring** materially helps survival but sits **below relief/blended** in this calibration. |
| **Survival** | Improvement in **business survival** over ~5–10 years | **Answered in proxy form:** isolated **cohort survival** after 60 months (no new entrants in that sub-run) tracks **~37% → ~41–42%** for the best bundles; aligns in order of magnitude with **national ONS ~38%** five-year benchmark for a different population definition (§5.1). |
| **Employment** | **Employment growth** / jobs | **Not answered.** The state is a **count of businesses**, not jobs. Any sentence about jobs would require an **employment layer** (see §6). |
| **Sectors & business types** | **For which sectors** does each lever work | **Partially encoded, not separately reported:** portfolios include **sector-specific** multipliers (e.g. relief tilts hospitality/retail); evaluation outputs are **aggregated** over sectors. We have **not** published a ranking **by sector** or by size band. |
| **Economic conditions** | Performance under **recession vs expansion** | **Answered in stylised form:** three overlays on the same historical panel. **Rankings** (relief/blended best on survival; startup strong on stock) are **qualitatively stable**; levels shift modestly—no regime flip in this run. |
| **Displacement & additionality** | **Net** effect vs **moving** activity | **Not answered quantitatively.** `-displacement` is a **heuristic** formation leakage vs neighbours, not calibrated displacement rates from evaluation literature. |
| **Value for money** | **Cost per** survivor / job | **Not answered.** Budget fields on portfolios are **indicative** only; there is no **cost constraint** or **optimisation** over budget splits in the evaluator. |
| **Time horizon** | **5–10 years** | **Partially:** stock paths to **120 months**; survival metric is **5‑year cohort**. Ten-year survival or employment paths are **not** yet standard outputs. |

**Plain-language takeaway:** for this **Hull-shaped** calibration, if policymakers care most about **keeping young businesses alive through year five** (as the model defines it), **relief-heavy and blended** designs dominate **startup-only** stylisations; if they care most about **growing the register count**, **startup-heavy and blended** look stronger—but **jobs**, **true net additionality**, and **sector-by-sector winners** are still **outside** what the current outputs prove.

### 5.4 Interpretation caveats

- Outcomes are **model-based counterfactuals**, not ex-post programme evaluations.
- **Displacement** is a **sketch** (neighbour mean births), not a spatial general equilibrium.
- **Elasticities** from the panel are **mapped heuristically** when `-auto-elasticities` is used.
- Official **employment** and **cost-per-job** are **not** yet endogenous; extend the state or post-process if those become reporting requirements.

---

## 6. Future work — closing the gaps in the original question

Each item states **what we would add** and **which part of the research question** it would unlock.

| Extension | Enables answering… |
|-----------|---------------------|
| **Employment / labour state** — e.g. per-business weight from CH accounts bands or ONS jobs-to-business ratios; optional stochastic job growth. | *“Employment growth”* and rough **cost per incremental job** once paired with programme costs. |
| **Sector- and size-disaggregated reporting** — evaluate outputs stratified by `policy.SectorOrder` (and cohorts); tables and plots per sector/size. | *“For which sectors and business types?”* |
| **Displacement & additionality** — multi-LA coupled model or **calibrated** leakage parameters from quasi-experimental anchors; neighbour policy reactions. | *“Net effect vs relocation”* and narrative comparability to **Enterprise Zone**-style evidence. |
| **Budgeted optimisation** — search or MPC over portfolio weights under a **spend cap**; tie `BudgetGBP` to WTP per survival point. | *“Greatest improvement under a fixed budget”* and **value-for-money** rankings. |
| **Longer horizon & richer scenarios** — 10-year survival, alternate rate/claim paths, stress from **SMC** posterior draws. | *“5–10 years”* and **deeper** macro robustness. |
| **Filing-based distress** — late accounts, dormancy flags into `distress_hazard_boost` or dedicated partition (**stochadex** `general` / `inference` pattern). | *“Early intervention”* and sharper **distress** channel than claimants alone. |
| **Full UK scale** — batch all LAs, standardised JSON + dashboard (`evalplot` or service). | *National coverage* and **spatial** policy comparisons. |
| **Networks, net zero, flags, international** — supply-chain shocks from CH graph; sector **transition** hazards; cross-country harmonisation. | Broader **Phase 5** questions beyond the core UK LA brief. |

This list is the intentional bridge from **what the repository demonstrates today** to the **full** policy question in §1.

---

## 7. Key data sources

| Source | Role |
|--------|------|
| [Companies House bulk company data](https://download.companieshouse.gov.uk/en_output.html) | Live register — formations and sector codes. |
| [ONS business demography](https://www.ons.gov.uk/) | Survival, births, deaths by LA / sector / cohort. |
| [NOMIS](https://www.nomisweb.co.uk/) | Claimant count and labour market context by LA. |
| [Bank of England statistics](https://www.bankofengland.co.uk/statistics) | Bank Rate. |
| [ONS NSPL](https://www.ons.gov.uk/methodology/geography/geographicalproducts/namesandcodes) | Postcode → local authority for CH join. |

Further citations (Enterprise Zone evaluations, rates consultations, etc.) remain valuable background for priors—see `pkg/policy/LiteraturePriorsTable` in code.

---

## 8. Contributing

Conventions for stochadex iterations, YAML config, and tests are in **`CLAUDE.md`**. For iteration implementations, follow **`simulator.Iteration`**: **`Configure`** once, **`Iterate`** must not mutate `params`, return state of width **`StateWidth`**.
