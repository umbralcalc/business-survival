package lifecycle

// AgeHistogram counts companies by age-in-months bucket. Buckets are 0..59
// and 60 lumps 60+ months (same convention as the monthly population model).
const AgeBucketCount = 61

func ageBucket(months int) int {
	if months < 0 {
		return 0
	}
	if months >= AgeBucketCount-1 {
		return AgeBucketCount - 1
	}
	return months
}

// SectorAgeHistogram accumulates cross-sectional ages at snapshot for one LA.
type SectorAgeHistogram struct {
	BySector map[string][AgeBucketCount]int
	NRows    int
}

// NewSectorAgeHistogram returns an empty accumulator.
func NewSectorAgeHistogram() *SectorAgeHistogram {
	return &SectorAgeHistogram{
		BySector: make(map[string][AgeBucketCount]int),
	}
}

// Add incorporates one parsed company (live or dissolved before snapshot age
// still has AgeMonthsSnapshot defined).
func (h *SectorAgeHistogram) Add(rec CompanyLifecycle) {
	h.NRows++
	b := ageBucket(rec.AgeMonthsSnapshot)
	arr := h.BySector[rec.Sector]
	arr[b]++
	h.BySector[rec.Sector] = arr
}

// LiveSectorMix returns sector → share of rows (for hazard-pool weighting).
func (h *SectorAgeHistogram) LiveSectorMix() map[string]float64 {
	if h.NRows == 0 {
		return nil
	}
	out := make(map[string]float64)
	for sec, arr := range h.BySector {
		n := 0
		for _, c := range arr {
			n += c
		}
		if n > 0 {
			out[sec] = float64(n) / float64(h.NRows)
		}
	}
	return out
}
