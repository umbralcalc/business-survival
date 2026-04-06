package geo

// AdjacentAuthorities lists ONS LA codes treated as primary spillover neighbours
// for stylised displacement (formation leakage). Expand as needed.
var AdjacentAuthorities = map[string][]string{
	"E06000010": {"E06000011"}, // Kingston upon Hull → East Riding of Yorkshire
	"E08000039": {"E07000064"}, // Sheffield → Rotherham (simplified ring)
	"E08000003": {"E08000004"}, // Manchester → Salford (simplified)
	"E09000030": {"E09000001"}, // Tower Hamlets → City of London (illustrative)
	"E09000033": {"E09000014"}, // Westminster → Camden
	"E07000008": {"E07000012"}, // Cambridge → South Cambridgeshire
	"E07000178": {"E07000180"}, // Oxford → Vale of White Horse (illustrative)
	"E06000052": {"E06000026"}, // Cornwall → Plymouth (coastal spillover sketch)
	"E06000014": {"E07000193"}, // York → Selby/Moor partial
	"E07000117": {"E07000124"}, // Burnley → Pendle
}
