package lifecycle

import (
	"strings"
	"time"
)

// ParseCHDate parses Companies House CSV dates ("02/01/2006").
func ParseCHDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("02/01/2006", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ParseRow extracts a lifecycle record if the row passes the same Ltd filters
// as cmd/explore and maps to one of the allowed LA codes.
//
// snapshot is the as-of date for AgeMonthsSnapshot and censoring (e.g. CSV
// publication date).
func ParseRow(row []string, laCode string, snapshot time.Time) (CompanyLifecycle, bool) {
	if len(row) <= ColSICPrimaryText {
		return CompanyLifecycle{}, false
	}
	cat := strings.TrimSpace(row[ColCompanyCategory])
	if cat != "Private Limited Company" && cat != "Public Limited Company" {
		return CompanyLifecycle{}, false
	}
	incorp, ok := ParseCHDate(row[ColIncorporationDate])
	if !ok {
		return CompanyLifecycle{}, false
	}
	if incorp.After(snapshot) {
		return CompanyLifecycle{}, false
	}

	var diss *time.Time
	if d, ok := ParseCHDate(row[ColDissolutionDate]); ok && !d.IsZero() {
		dCopy := d
		diss = &dCopy
	}

	st := strings.TrimSpace(row[ColCompanyStatus])
	ageThrough := snapshot
	if diss != nil && !diss.After(snapshot) {
		ageThrough = *diss
	}
	age := MonthsBetweenCalendar(incorp, ageThrough)
	if age < 0 {
		age = 0
	}

	censored := diss == nil || diss.After(snapshot)

	cl := CompanyLifecycle{
		CompanyNumber:     strings.TrimSpace(row[ColCompanyNumber]),
		AreaCode:          laCode,
		Sector:            SectorFromSIC(row[ColSICPrimaryText]),
		Incorporation:     incorp,
		Dissolution:       diss,
		CompanyStatus:     st,
		AgeMonthsSnapshot: age,
		Censored:          censored,
	}
	return cl, true
}
