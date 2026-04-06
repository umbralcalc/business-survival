package lifecycle

import "time"

// CompanyLifecycle summarises one row of the Companies House bulk product
// at a fixed CSV snapshot date (live register).
type CompanyLifecycle struct {
	CompanyNumber    string
	AreaCode         string
	Sector           string
	Incorporation    time.Time
	Dissolution      *time.Time
	CompanyStatus    string
	AgeMonthsSnapshot int
	// Censored is true when no dissolution is observed on or before the snapshot
	// (right-censored lifetime for survival analysis).
	Censored bool
}

// MonthsBetweenCalendar counts whole calendar months from start (inclusive of
// start month) to end, treating both as year-month anchors.
func MonthsBetweenCalendar(start, end time.Time) int {
	y := end.Year() - start.Year()
	m := int(end.Month()) - int(start.Month())
	return y*12 + m
}
