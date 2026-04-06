package policy

import "maps"

// MergeParamMaps overlays override into base (shallow copy of base first).
func MergeParamMaps(base, override map[string][]float64) map[string][]float64 {
	out := make(map[string][]float64, len(base)+len(override))
	maps.Copy(out, base)
	for k, v := range override {
		cp := make([]float64, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// PortfolioParams returns simulator param entries for portfolio levers.
// Baseline returns nil (no policy keys — population defaults apply).
func PortfolioParams(p Portfolio) map[string][]float64 {
	if p.ID == "baseline" {
		return nil
	}
	birth := p.BirthScale
	if birth <= 0 {
		birth = 1.0
	}
	death := p.DeathHazardScale
	if death <= 0 {
		death = 1.0
	}
	infant := p.InfantHazardScale
	if infant <= 0 {
		infant = 1.0
	}
	out := map[string][]float64{
		"policy_birth_scale":          {birth},
		"policy_death_hazard_scale":   {death},
		"policy_infant_hazard_scale":  {infant},
	}

	n := len(SectorOrder)
	sb := make([]float64, n)
	sh := make([]float64, n)
	needB, needH := false, false
	for i, name := range SectorOrder {
		sb[i] = 1.0
		sh[i] = 1.0
		if p.SectorBirthScale != nil {
			if v, ok := p.SectorBirthScale[name]; ok && v > 0 {
				sb[i] = v
				needB = true
			}
		}
		if p.SectorHazardScale != nil {
			if v, ok := p.SectorHazardScale[name]; ok && v > 0 {
				sh[i] = v
				needH = true
			}
		}
	}
	if needB {
		out["policy_sector_birth_scale"] = sb
	}
	if needH {
		out["policy_sector_hazard_scale"] = sh
	}
	return out
}
