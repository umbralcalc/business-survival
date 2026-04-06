package calibrate

// COVIDBirthSlump checks the natural-experiment pattern: hospitality formations
// fall more sharply than technology in the first locked-down month vs prior year.
func COVIDBirthSlump(monthly map[string]map[string]int) (hospitalityYoY, technologyYoY float64) {
	hospitalityYoY = YearOverYearRatio(monthly, "2020-04", "2019-04", "Hospitality")
	technologyYoY = YearOverYearRatio(monthly, "2020-04", "2019-04", "Technology")
	return hospitalityYoY, technologyYoY
}
