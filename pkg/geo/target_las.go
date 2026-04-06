package geo

// TargetLAs is the set of 10 local authorities used for the initial Phase 1
// exploration. They span a deliberate range of economic contexts:
//
//   - High-startup London boroughs (Tower Hamlets, Westminster)
//   - Core northern cities (Manchester, Sheffield)
//   - Major ex-industrial / Enterprise Zone hosts (Hull, Burnley)
//   - University-driven economies (Cambridge, Oxford)
//   - Rural / peripheral (Cornwall)
//   - Historic / services (York)
var TargetLAs = map[string]string{
	"E09000030": "Tower Hamlets",
	"E09000033": "Westminster",
	"E08000003": "Manchester",
	"E08000039": "Sheffield",
	"E06000010": "Kingston upon Hull, City of",
	"E07000117": "Burnley",
	"E07000008": "Cambridge",
	"E07000178": "Oxford",
	"E06000052": "Cornwall",
	"E06000014": "York",
}

// LegacyCodeAliases maps current (post-reorganisation) LA codes to historical
// codes that still appear in other datasets such as NOMIS. Sheffield was
// renumbered from E08000019 to E08000039 for the 2025 geography refresh, but
// NOMIS claimant count series still carry the old code.
var LegacyCodeAliases = map[string][]string{
	"E08000039": {"E08000019"}, // Sheffield
}

// AllCodesForLA returns the current code followed by any legacy aliases.
func AllCodesForLA(code string) []string {
	return append([]string{code}, LegacyCodeAliases[code]...)
}

// TargetLAFilter returns a set-membership map for LoadNSPL.
func TargetLAFilter() map[string]bool {
	f := make(map[string]bool, len(TargetLAs))
	for code := range TargetLAs {
		f[code] = true
	}
	return f
}
