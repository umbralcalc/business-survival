package lifecycle

import "strings"

// SectorFromSIC maps the first two digits of an ONS SIC code text field to a
// coarse sector group aligned with cmd/explore and Phase 1 analysis.
func SectorFromSIC(sic string) string {
	sic = strings.TrimSpace(sic)
	if len(sic) < 2 {
		return "Other"
	}
	prefix := sic[:2]
	switch prefix {
	case "41", "42", "43":
		return "Construction"
	case "47":
		return "Retail"
	case "55", "56":
		return "Hospitality"
	case "62", "63":
		return "Technology"
	case "69", "70", "71", "72", "73", "74":
		return "Professional"
	default:
		return "Other"
	}
}
