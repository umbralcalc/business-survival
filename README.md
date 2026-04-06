# Small Business Survival & Support Policy Simulation: Project Plan

## Applying the Stochadex to Business Support Intervention Optimisation

---

## Overview

Build a stochastic simulation of small business lifecycle dynamics — formation, growth, distress, and failure — learned from freely available Companies House, ONS business demography, and economic data, with a decision science layer to evaluate and compare business support interventions in terms of their impact on survival probabilities, employment, and local economic resilience.

The core question: **given the current economic conditions and business population in a region, which combination of support interventions (rate relief, grants, incubator programmes, mentoring) produces the greatest improvement in small business survival and employment growth over 5–10 years, and for which sectors and business types?**

---

## Why This Problem

- The UK had approximately 2.9 million VAT/PAYE registered businesses on the Inter-Departmental Business Register (IDBR). Hundreds of thousands are born and die each year — in 2020 alone, 358,000 births and 316,000 deaths.
- Small business failure is inherently stochastic — survival depends on sector, location, economic cycle, access to finance, founder characteristics, and luck. Yet policy interventions are evaluated almost entirely with deterministic before/after comparisons or simple regression.
- Enterprise Zones, the flagship place-based business support policy, have been evaluated by the Centre for Cities and found to have created jobs at a cost of approximately £28,540 per job — but with major questions about displacement (did businesses just move from nearby?) and additionality (would they have formed anyway?). These are fundamentally counterfactual questions.
- The government is actively rethinking business rates policy — a Call for Evidence was published in late 2025 on how business rates influence investment decisions, and Small Business Rates Relief currently provides 100% relief for properties with rateable values under £12,000. Whether these reliefs actually improve survival is empirically unresolved.
- Almost nobody models business demography stochastically. The data is surprisingly rich — Companies House publishes incorporation and dissolution data for every UK company, and ONS publishes survival curves by sector, region, and cohort — but it's treated as descriptive statistics, not as a dynamical system to simulate and intervene on.

---

## The Gap This Fills

| Approach | Examples | Limitation |
|----------|----------|------------|
| Descriptive business demography | ONS Business Demography bulletins | Reports birth/death rates and survival curves but doesn't model dynamics or evaluate interventions |
| Programme evaluation (quasi-experimental) | Centre for Cities Enterprise Zone evaluation, UEZ evaluation | Estimates average treatment effects but can't simulate alternative policy designs or predict outcomes in new contexts |
| Econometric survival analysis | Academic Cox proportional hazards models of firm failure | Static: estimates hazard ratios from historical data, doesn't simulate forward under policy counterfactuals |
| Agent-based business models | Academic models of firm ecosystems | Theoretically rich but rarely calibrated to real registration data at scale |

**The stochadex differentiator:** a stochastic simulation of the full business lifecycle — birth, growth, distress, death — learned from millions of Companies House records and ONS demography data, with a decision science layer that evaluates support interventions by simulating their effect on transition probabilities. Same proven pattern: ingest freely available data, build a simulation that learns from it, optimise policy actions.

---

## Phase 1: Data Ingestion

### 1.1 Company registration data

**Source: Companies House Free Company Data Product**

- Bulk CSV snapshot of all live companies on the register
- Fields: company number, name, incorporation date, dissolution date, company status, company category, registered office address (including postcode), up to 4 SIC codes, accounts category, confirmation statement date, previous names
- Updated monthly (end of previous month, available within 5 working days)
- Free download, no registration required

**Download:** `download.companieshouse.gov.uk/en_output.html`

**Source: Companies House Accounts Data Product**

- Individual company accounts filed electronically, in XBRL format
- Daily and monthly bulk downloads
- Contains financial data (turnover, profit/loss, assets, liabilities, employee count) for companies that file detailed accounts
- Free download

**Source: Companies House API**

- RESTful API for individual company lookups, officer searches, filing history
- Free with registration (API key)
- Useful for enriching bulk data with filing history and officer details

### 1.2 Business demography data

**Source: ONS Business Demography**

- Annual publication: births, deaths, survivals, and active stock of UK enterprises
- Breakdowns by SIC 2007 industry group, region, local authority
- Survival rates tracked for up to 5 years after birth by cohort
- Employer vs. non-employer business demography
- High-growth business identification (>20% employment growth per annum over 3 years)
- Based on the Inter-Departmental Business Register (IDBR) — all businesses registered for VAT and/or PAYE
- Free download from ONS

**Key datasets:**
- Business demography reference table (births, deaths, active, survival by LA and SIC)
- Employer business demography (subset with ≥1 employee)
- Multiple business registrations at a single postcode (data quality flag)

### 1.3 Economic context data

**Source: ONS / NOMIS Labour Market Statistics**

- Employment rate, claimant count, job vacancies by travel-to-work area and local authority
- Sector composition of local employment
- Quarterly and monthly
- The demand-side driver of business survival

**Source: Bank of England**

- Bank Rate, business lending data, credit conditions survey
- Interest rates and credit availability directly affect small business survival — rate rises increase debt service costs and reduce consumer spending

**Source: ONS Regional GDP and GVA**

- Gross Value Added by local authority and industry
- The output-side context for business formation and growth

### 1.4 Business support intervention data

**Source: VOA Non-Domestic Rating (Business Rates)**

- Rateable values and relief data by local authority
- Small Business Rates Relief coverage
- Enterprise Zone rate relief uptake

**Source: MHCLG / DLUHC Enterprise Zone Data**

- Employment and business counts in Enterprise Zones
- Rate relief and capital allowances distributed
- Published in evaluation reports and live tables

**Source: British Business Bank**

- Data on government-backed lending schemes (Start Up Loans, CBILS, Recovery Loan Scheme)
- Regional breakdowns of lending volumes
- Published in annual reports and data releases

**Source: BEIS / DBT Business Support Evaluations**

- Published evaluations of programmes: GrowthAccelerator, Start Up Loans, Innovate UK grants, University Enterprise Zones
- Contain estimated effect sizes (survival, employment, GVA impacts) that can serve as prior distributions for the stochadex

### 1.5 Initial data scope

- **Geography:** 10–20 local authorities spanning different economic contexts — e.g., a high-startup-rate London borough (Tower Hamlets), a Northern industrial town (Burnley), a university city (Cambridge), an Enterprise Zone host (Sheffield, Humber), a rural area (Cornwall)
- **Time window:** Companies House data from 2000–2025 for lifecycle analysis; ONS demography from 2009–2024
- **Sectors:** Focus on sectors with high birth/death rates: professional services (SIC 69–74), retail (SIC 47), hospitality (SIC 55–56), construction (SIC 41–43), technology (SIC 62–63)
- **Resolution:** Monthly for incorporation/dissolution events, annual for ONS demography and survival curves, quarterly for economic context

---

## Phase 2: Model Structure

### 2.1 State variables

The stochadex simulation tracks a regional business population as a coupled stochastic system:

1. **Business birth process** — stochastic, driven by economic conditions (GDP growth, employment, credit availability), local entrepreneurial culture, and sector-specific factors. New businesses enter with characteristics (sector, size, location).
2. **Growth/stasis process** — stochastic transitions between size bands (micro → small → medium), with transition rates depending on sector, age, economic conditions, and access to support.
3. **Distress process** — stochastic entry into financial difficulty (late filing, dormancy, CCJ), with rates depending on economic shocks, sector conditions, interest rates, and business characteristics.
4. **Death process** — stochastic dissolution/liquidation, with hazard rates depending on age (strong age-dependence in first 3 years), sector, economic conditions, and whether support interventions have been received.
5. **Employment process** — stochastic headcount evolution within surviving businesses, the key output metric alongside survival.

### 2.2 Simulation diagram

```
┌─────────────────────────────────────────────────────────┐
│             MACROECONOMIC ENVIRONMENT                    │
│  GDP growth, interest rates, credit conditions,          │
│  consumer confidence, sector-specific demand             │
│  (stochastic, learned from ONS/BoE data)                │
└───┬──────────────┬─────────────┬────────────────────────┘
    │              │             │
    ▼              ▼             ▼
┌─────────────────────────────────────────────────────────┐
│              BUSINESS BIRTH PROCESS                       │
│  New incorporations at Companies House                   │
│  Rate = f(economic conditions, sector, region)           │
│  Each birth has: sector (SIC), location, type            │
│  INTERVENTION: Startup grants, incubators, mentoring     │
└──────────────────┬──────────────────────────────────────┘
                   │ new businesses enter population
                   ▼
┌─────────────────────────────────────────────────────────┐
│           ACTIVE BUSINESS POPULATION                      │
│  State per business: age, sector, size band, status      │
│                                                          │
│  ┌─────────┐    ┌──────────┐    ┌──────────┐           │
│  │  MICRO  │───▶│  SMALL   │───▶│  MEDIUM  │           │
│  │ (0-9)   │    │ (10-49)  │    │ (50-249) │           │
│  └────┬────┘    └────┬─────┘    └────┬─────┘           │
│       │              │               │                   │
│       ▼              ▼               ▼                   │
│  ┌──────────────────────────────────────────┐           │
│  │           DISTRESS STATE                  │           │
│  │  Late filing, dormancy, CCJs              │           │
│  │  INTERVENTION: Rate relief, rescue advice │           │
│  └─────────────────┬────────────────────────┘           │
│                    │                                     │
│                    ▼                                     │
│  ┌──────────────────────────────────────────┐           │
│  │           DEATH (DISSOLUTION)             │           │
│  │  Voluntary strike-off, liquidation,       │           │
│  │  compulsory dissolution                   │           │
│  └──────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│         SUPPORT INTERVENTIONS (POLICY LEVERS)            │
│                                                          │
│  Pre-birth: Enterprise education, startup loans          │
│  Early-stage: Incubators, mentoring, grant programmes    │
│  Growth: Rate relief, R&D tax credits, export support    │
│  Distress: Business rescue advice, rate hardship relief  │
│  Place-based: Enterprise Zones, Investment Zones         │
│                                                          │
│  Each intervention modifies specific transition rates    │
│  Effect magnitudes are uncertain — modelled as priors    │
│  from published evaluation evidence                      │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│              OUTCOMES                                     │
│  Business survival rate (1yr, 3yr, 5yr)                  │
│  Net business stock change                               │
│  Total employment in surviving businesses                │
│  GVA contribution                                        │
│  Displacement: did support create new activity or just   │
│  move it from elsewhere?                                 │
└─────────────────────────────────────────────────────────┘
```

### 2.3 Key modelling choices

- **Local authority level** as the primary spatial unit, matching ONS business demography and Companies House postcode data.
- **Monthly time step** for birth/death events (matching Companies House update cadence), with annual calibration against ONS demography cohort survival curves.
- **Age-dependent hazard:** Business failure risk is strongly age-dependent — highest in years 1–3, declining with maturity. The stochadex models this as a time-inhomogeneous hazard function learned from ONS survival curves by sector and region.
- **Sector heterogeneity:** Different SIC groups have very different birth rates, death rates, and survival profiles. Hospitality has much higher churn than professional services. The model learns separate dynamics per sector group.
- **Intervention effects as uncertain parameter modifiers:** Each support intervention modifies specific transition rates (e.g., rate relief reduces the distress→death transition probability). The magnitude is uncertain — modelled as a prior distribution informed by published evaluation evidence, updated with local data where available.
- **Ensemble approach:** Run hundreds of stochastic trajectories per intervention portfolio to build distributions of survival and employment outcomes.

---

## Phase 3: Learning from Data

### 3.1 Simulation-based inference

1. **Construct lifecycle histories** from Companies House bulk data: for each company, extract incorporation date, dissolution date (if applicable), SIC codes, registered address postcode, and accounts category. This gives millions of observed lifetimes and censored observations.
2. **Calibrate against ONS business demography:** The ONS survival curves by cohort, sector, and region provide the aggregate targets. The stochadex learns micro-level transition rates that reproduce these macro-level survival statistics.
3. **Fit economic sensitivity:** How do birth and death rates co-move with GDP growth, interest rates, and local employment? Learn these elasticities from the joint time series.
4. **Key parameters to learn:**
   - Baseline hazard function by sector and age (the "survival curve shape")
   - Economic sensitivity: how much do recessions increase death rates and suppress birth rates?
   - Interest rate sensitivity: how does the BoE rate affect small business survival (debt service channel)?
   - Regional variation: conditional on sector and economic conditions, how much do survival rates differ across regions?
   - Growth transition probabilities: what fraction of micro businesses reach small/medium scale, by sector and age?

### 3.2 Intervention effect estimation

For each support intervention, construct a prior distribution of its effect on survival/growth from published evaluation evidence:

| Intervention | Effect parameter | Prior source |
|-------------|-----------------|--------------|
| Enterprise Zone rate relief | Effect on business birth rate in zone | Centre for Cities evaluation: 13,500 jobs at £28,540/job, but with significant displacement |
| Small Business Rates Relief | Effect on distress→death transition | BEIS/government Call for Evidence (2025): qualitative evidence that cliff-edges deter expansion |
| Start Up Loans | Effect on 1-year survival of recipients | British Business Bank evaluations |
| Incubator/accelerator programmes | Effect on growth transition rate | UEZ evaluation: £100K additional GVA per programme, but diminishing over time |
| Mentoring programmes | Effect on survival and growth | BEIS mentoring evidence review |
| R&D tax credits / Innovate UK grants | Effect on growth and high-growth probability | Innovate UK impact evaluations |

Where possible, update these priors with local data from specific intervention areas.

### 3.3 The displacement problem

The central methodological challenge: did the Enterprise Zone *create* businesses, or did it just *move* them from nearby areas? The stochadex addresses this by modelling the coupled dynamics of the zone and its hinterland — if births increase in the zone but decrease in adjacent LAs by the same amount, the net effect is zero. This requires fitting a spatial model where business formation in one LA can be affected by conditions (including policy) in neighbouring LAs.

### 3.4 Validation strategy

- **Temporal holdout:** Train on 2009–2020, predict 2021–2025 business demography (a demanding test given COVID and the rate-rise cycle).
- **COVID natural experiment:** The pandemic caused massive, sector-differentiated business distress. Can the model reproduce the observed pattern of deaths by sector (hospitality devastated, technology resilient)?
- **Cohort survival:** Does the model reproduce the ONS 1yr, 3yr, and 5yr survival curves for each sector and region?
- **Cross-LA validation:** Train on a subset of local authorities, predict demography in held-out LAs.

---

## Phase 4: Decision Science Layer

### 4.1 Policy actions to evaluate

| Policy type | How it acts in the model | Decision variables |
|-------------|--------------------------|-------------------|
| **Small Business Rates Relief** | Reduces distress probability for micro businesses | Threshold (rateable value), taper rate |
| **Enterprise Zone designation** | Reduces rates, increases birth rate in zone (but may displace from nearby) | Zone location, duration, rate relief level |
| **Startup grants/loans** | Increases early-stage survival and reduces capital constraints | Grant size, eligibility criteria, sector targeting |
| **Incubator/accelerator** | Increases growth transition probability for participants | Programme intensity, sector focus, duration |
| **Mentoring programmes** | Reduces death rate for participants, increases growth | Coverage (% of eligible businesses reached), matching quality |
| **Sector-specific support** | Targeted interventions for high-potential or vulnerable sectors | Sector selection, intervention type, budget allocation |
| **Portfolio approach** | Combination of multiple interventions targeting different lifecycle stages | Budget allocation across programmes |

### 4.2 The lifecycle targeting question

A key insight is that different interventions are effective at different lifecycle stages. Rate relief helps established businesses in distress; startup loans help nascent businesses survive year 1; incubators help young businesses grow. The stochadex can evaluate *portfolios* that target different stages, answering: "given a fixed budget, should we spend 60% on rate relief and 40% on startup loans, or 30/30/40 split with incubator funding?"

### 4.3 Objective function

For each intervention portfolio, simulate multiple trajectories across economic scenarios and evaluate:

- **Primary outcome:** Expected 5-year business survival rate for the cohort of businesses born in year 1 of the intervention
- **Employment outcome:** Expected net employment in surviving businesses at 5 years
- **Value for money:** Cost per additional surviving business and cost per additional job (comparable to Enterprise Zone evaluations)
- **Displacement metric:** Net effect across the LA and its neighbours (is the intervention creating activity or moving it?)
- **Robustness:** Performance under recession and expansion scenarios
- **Distributional:** Which sectors and business types benefit most?

### 4.4 Output

For a given local authority and budget, produce actionable recommendations:

> *"For Sheffield (current 5-year business survival rate: 42%), a portfolio of £5M over 5 years split between targeted rate relief for hospitality businesses in years 1–2 (£2M), an incubator programme for tech startups (£1.5M), and a mentoring network for businesses in years 2–4 (£1.5M) increases the expected 5-year survival rate to 46.8% (90% CI: 44.1% to 49.5%), creating an estimated 340 additional surviving businesses and 1,200 additional jobs. This compares to Enterprise Zone rate relief alone, which would increase the birth rate by 8% but with 60% of the effect being displacement from adjacent areas, yielding only 130 net additional surviving businesses. Under a recession scenario (GDP −2%), the portfolio outperforms rate relief alone by a wider margin, as the mentoring component provides resilience that pure cost reduction does not."*

---

## Phase 5: Extensions

1. **Full UK coverage:** Scale to all ~370 local authorities, producing a national business support dashboard that government departments could use to allocate regional funding
2. **Supply chain dynamics:** Model inter-business dependencies — when a key customer or supplier fails, it increases the hazard rate for connected businesses. Use Companies House officer and significant person data to infer network connections.
3. **Net zero transition:** Model the differential impact of decarbonisation on business survival by sector — fossil-fuel-dependent businesses face declining demand, while green economy businesses may have higher birth rates but uncertain survival. Evaluate whether targeted support can smooth the transition.
4. **Real-time distress monitoring:** Use Companies House filing patterns (late accounts, dormancy flags) as leading indicators of business distress, connected to the simulation for early intervention targeting.
5. **International comparison:** Adapt the model to other jurisdictions with comparable open data (e.g., Ireland's CRO, Netherlands' KvK) for cross-country policy comparison.
6. **Founder characteristics:** Where Companies House officer data permits, model how director experience, serial entrepreneurship, and team composition affect survival — enabling more targeted mentoring and support matching.

---

## Concrete First Steps

### Week 1–2: Data acquisition and exploration ✅

- [x] Download Companies House Free Company Data Product (March 2026 snapshot, 5.26M live limited companies)
- [x] Download ONS Business Demography reference table (2024 publication — 2,126 survival series, 1,688 birth/death rows by LA)
- [x] Download NOMIS claimant count for all 406 LAs, monthly 2013–2026 (64k rows)
- [x] Download Bank of England Bank Rate daily series from 2000 (via stats database CSV API)
- [x] Download ONS NSPL November 2025 postcode → LA lookup (2.5M postcodes)
- [x] Select 10 target local authorities: Westminster, Tower Hamlets, Manchester, Sheffield, Cornwall, Cambridge, Oxford, York, Kingston upon Hull, Burnley (see `pkg/geo/target_las.go`)
- [x] Exploratory analysis pipeline:
  - `cmd/parse-ons` → `dat/ons_demography.json` (ONS survival curves by LA)
  - `cmd/explore` → `dat/la_births.json` (per-LA monthly birth counts by sector group, enriched via postcode→LA join)
  - `cmd/analyse` → `dat/la_panel.json` (joined monthly panel with BoE rate + claimant count, plus Pearson correlations)
- [x] First-pass findings: 379k of 5.26M live companies are in the 10 target LAs (~7%); UK 5-year survival for the 2019 cohort is 38.4% (ONS). Raw ρ(births, rate) ≈ 0.75 and ρ(births, claimant) ≈ 0.4 across LAs, but this is dominated by a shared upward trend over 2013–2025 — proper elasticity estimation requires detrending/differencing and belongs to Phase 3.

### Week 3–4: Minimal stochadex simulation

- [ ] Implement a single-LA business population model with birth and death processes
- [ ] Parameterise with age-dependent hazard functions by sector
- [ ] Add economic sensitivity (GDP, interest rate covariates)
- [ ] Verify the simulation reproduces ONS survival curves and aggregate birth/death rates

### Week 5–6: Simulation-based inference

- [ ] Construct company lifecycle histories from Companies House data
- [ ] Set up SBI to learn transition rate parameters from observed lifecycle data, conditional on economic covariates
- [ ] Validate: does the model reproduce the COVID business death pattern by sector?
- [ ] Fit the economic sensitivity elasticities using the 2008–2009 and 2020 recessions as natural experiments

### Week 7–8: Decision science layer

- [ ] Implement 3–4 candidate support intervention portfolios as action sets
- [ ] Set intervention effect priors from published evaluation literature
- [ ] Run policy evaluation across economic scenarios (baseline, recession, expansion)
- [ ] Produce initial findings and visualisations for target LAs
- [ ] Write up as a blog post in the "Engineering Smart Actions in Practice" series

---

## Key Data Sources Summary

| Source | URL | Data type | Access |
|--------|-----|-----------|--------|
| Companies House Free Company Data | download.companieshouse.gov.uk | Bulk CSV: all live companies with incorporation/dissolution dates, SIC codes, address, status | Free monthly download |
| Companies House Accounts Data | download.companieshouse.gov.uk/en_accountsdata.html | Company accounts in XBRL: financials, employee counts | Free daily/monthly download |
| Companies House API | developer.company-information.service.gov.uk | Individual company lookups, filing history, officer data | Free with API key registration |
| ONS Business Demography | ons.gov.uk (Business births, deaths and survival rates) | Annual: births, deaths, active stock, survival rates by LA, SIC, cohort | Free download |
| ONS UK Business: Activity, Size and Location | ons.gov.uk | Annual count of businesses by LA, SIC, size band, legal form | Free download |
| NOMIS Labour Market Statistics | nomisweb.co.uk | Employment, claimant count, vacancies, sector composition by LA/TTWA | Free download |
| Bank of England | bankofengland.co.uk/statistics | Bank Rate, business lending volumes, credit conditions | Free download |
| ONS Regional GVA | ons.gov.uk | Gross Value Added by LA and industry | Free download |
| VOA Non-Domestic Rating | gov.uk (search "non-domestic rating") | Rateable values, business rates relief by LA | Free download |
| British Business Bank | british-business-bank.co.uk | Start Up Loans data, government-backed lending by region | Annual reports, some open data |
| BEIS/DBT Evaluations | gov.uk (search programme names) | Published evaluation evidence for business support programmes | Free PDF reports |

---

## References and Related Work

- ONS Business Demography UK: 2022 — annual publication identifying births, deaths, survivals, and high-growth businesses, based on the IDBR (VAT/PAYE registrations)
- Centre for Cities "In the Zone?" (2019) — evaluation finding Enterprise Zones created 13,500 jobs at £28,540 per job, with city-centre zones vastly outperforming suburban/rural ones and significant displacement concerns
- UEZ Final Evaluation (2025) — University Enterprise Zones delivered £100K additional GVA and created jobs, outperforming comparators, but with diminishing engagement effects over time
- BEIS Business Rates Call for Evidence (2025) — government consultation on how rates influence investment, highlighting the SBRR cliff-edge problem and exploring improvement relief and growth accelerator designs
- Companies House URI Customer Guide — technical documentation for the bulk data fields and linked data SPARQL endpoint
- ONS Multiple Business Registrations explainer — important data quality note on artificial inflation of birth/death counts from multiple registrations at single postcodes