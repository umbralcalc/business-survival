package lifecycle

import (
	"testing"
	"time"

	"gonum.org/v1/gonum/floats/scalar"
)

func TestSectorFromSIC(t *testing.T) {
	if SectorFromSIC("55100 - Hotels") != "Hospitality" {
		t.Fatal(SectorFromSIC("55100 - Hotels"))
	}
	if SectorFromSIC("62020 - Computer consultancy") != "Technology" {
		t.Fatal()
	}
}

func TestParseRow_SampleCSV(t *testing.T) {
	snap := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	row := []string{
		"TestCo", "123", "", "", "1 High St", "", "London", "", "UK", "E1 6AN",
		"Private Limited Company", "Active", "United Kingdom", "", "01/06/2020",
		"31", "12", "", "", "FULL", "", "", "", "", "", "",
		"56101 - Licensed restaurants", "", "", "", "", "", "", "",
	}
	rec, ok := ParseRow(row, "E09000030", snap)
	if !ok {
		t.Fatal("expected ok")
	}
	if rec.Sector != "Hospitality" || !rec.Censored {
		t.Fatalf("%+v", rec)
	}
	wantAge := MonthsBetweenCalendar(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC), snap)
	if rec.AgeMonthsSnapshot != wantAge {
		t.Fatalf("age got %d want %d", rec.AgeMonthsSnapshot, wantAge)
	}
}

func TestSectorAgeHistogram(t *testing.T) {
	h := NewSectorAgeHistogram()
	h.Add(CompanyLifecycle{Sector: "Retail", AgeMonthsSnapshot: 5})
	h.Add(CompanyLifecycle{Sector: "Retail", AgeMonthsSnapshot: 70})
	mix := h.LiveSectorMix()
	if !scalar.EqualWithinAbs(mix["Retail"], 1.0, 1e-9) {
		t.Fatalf("%v", mix)
	}
	if h.BySector["Retail"][5] != 1 || h.BySector["Retail"][60] != 1 {
		t.Fatalf("%v", h.BySector["Retail"])
	}
}
